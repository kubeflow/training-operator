/*
Copyright The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tlsconfig

import (
	"crypto/tls"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"

	configapi "github.com/kubeflow/trainer/v2/pkg/apis/config/v1alpha1"
)

// Apply configures c from the TLS options in the Configuration API. It is the
// single entry point for translating configapi.TLSOptions into a *tls.Config,
// so the metrics, webhook, and status servers all resolve TLS identically.
//
// A nil opts is valid and yields the secure defaults.
func Apply(c *tls.Config, opts *configapi.TLSOptions) {
	if opts != nil && len(opts.NextProtos) > 0 {
		c.NextProtos = opts.NextProtos
	} else {
		// HTTP/2 is disabled by default to mitigate the Rapid Reset CVEs
		// (CVE-2023-44487 and CVE-2023-39325). For more information see:
		// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
		// - https://github.com/advisories/GHSA-4374-p667-p6c8
		c.NextProtos = []string{"http/1.1"}
	}

	if opts == nil {
		return
	}

	if opts.MinVersion != "" {
		if v := parseTLSVersion(opts.MinVersion); v != 0 {
			c.MinVersion = v
		}
	}
	if len(opts.CipherSuites) > 0 {
		if ids := parseCipherSuiteIDs(opts.CipherSuites); len(ids) > 0 {
			c.CipherSuites = ids
		}
	}
	if len(opts.CurvePreferences) > 0 {
		if curves := parseCurveIDs(opts.CurvePreferences); len(curves) > 0 {
			c.CurvePreferences = curves
		}
	}
}

// parseTLSVersion converts a TLS version string to its uint16 constant.
// It logs a warning and returns 0 if the version is not recognized, which
// leaves the Go default in place rather than failing the server startup.
// Recognition is enforced earlier, at configuration load time, by Validate.
func parseTLSVersion(version string) uint16 {
	if v, ok := lookupTLSVersion(version); ok {
		return v
	}
	klog.Warningf("Unrecognized TLS version %q, using Go default", version)
	return 0
}

// lookupTLSVersion converts a TLS version string to its uint16 constant.
// The second return value reports whether the version is recognized.
func lookupTLSVersion(version string) (uint16, bool) {
	switch version {
	case configapi.TLSVersion10:
		return tls.VersionTLS10, true
	case configapi.TLSVersion11:
		return tls.VersionTLS11, true
	case configapi.TLSVersion12:
		return tls.VersionTLS12, true
	case configapi.TLSVersion13:
		return tls.VersionTLS13, true
	default:
		return 0, false
	}
}

// parseCipherSuiteIDs converts cipher suite name strings to their uint16 IDs.
// Unrecognized names are ignored with a warning. Insecure suites are accepted,
// since operators may still need them for legacy clients, but warn loudly.
// Recognition is enforced earlier, at configuration load time, by Validate.
func parseCipherSuiteIDs(names []string) []uint16 {
	ids := make([]uint16, 0, len(names))
	for _, name := range names {
		if isRC4Suite(name) {
			klog.Warningf("Cipher suite %q is prohibited by RFC 7465, ignoring", name)
			continue
		}
		if id, recognized, insecure := lookupCipherSuiteID(name); recognized {
			if insecure {
				klog.Warningf("Cipher suite %q is insecure and should not be used in production", name)
			}
			ids = append(ids, id)
		} else {
			klog.Warningf("Unrecognized cipher suite %q, ignoring", name)
		}
	}
	return ids
}

// lookupCipherSuiteID resolves a Go crypto/tls cipher suite name to its ID.
// recognized reports whether the name is known to this build; insecure
// reports whether the suite is on Go's insecure list.
func lookupCipherSuiteID(name string) (id uint16, recognized, insecure bool) {
	for _, cs := range tls.CipherSuites() {
		if cs.Name == name {
			return cs.ID, true, false
		}
	}
	for _, cs := range tls.InsecureCipherSuites() {
		if cs.Name == name {
			return cs.ID, true, true
		}
	}
	return 0, false, false
}

// isRC4Suite reports whether the name is one of Go's RC4 cipher suites.
// RC4 is negotiable by this build but categorically broken: RFC 7465
// prohibits it, and no OpenShift TLS profile contains it, so it is rejected
// outright rather than accepted with the legacy-suite warning.
func isRC4Suite(name string) bool {
	switch name {
	case "TLS_RSA_WITH_RC4_128_SHA", "TLS_ECDHE_RSA_WITH_RC4_128_SHA", "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":
		return true
	}
	return false
}

// supportedCurves maps Go crypto/tls curve names to their tls.CurveID values.
var supportedCurves = map[string]tls.CurveID{
	"CurveP256":          tls.CurveP256,
	"CurveP384":          tls.CurveP384,
	"CurveP521":          tls.CurveP521,
	"X25519":             tls.X25519,
	"X25519MLKEM768":     tls.X25519MLKEM768,
	"SecP256r1MLKEM768":  tls.SecP256r1MLKEM768,
	"SecP384r1MLKEM1024": tls.SecP384r1MLKEM1024,
}

// parseCurveIDs converts curve preference name strings to their tls.CurveID
// values. Unrecognized names are ignored with a warning, which leaves the Go
// default curve selection for them rather than failing the server startup.
// Recognition is enforced earlier, at configuration load time, by Validate.
func parseCurveIDs(names []string) []tls.CurveID {
	curves := make([]tls.CurveID, 0, len(names))
	for _, name := range names {
		if id, ok := supportedCurves[name]; ok {
			curves = append(curves, id)
		} else {
			klog.Warningf("Unrecognized TLS curve preference %q, ignoring", name)
		}
	}
	return curves
}

// Validate checks that the TLS option values are recognized by this build.
// It runs during configuration loading so that unrecognized values fail
// controller startup: the TLS option closures that call Apply cannot return
// errors, so recognition cannot be enforced at apply time.
func Validate(opts *configapi.TLSOptions, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if opts == nil {
		return allErrs
	}

	if opts.MinVersion != "" {
		if _, ok := lookupTLSVersion(opts.MinVersion); !ok {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("minVersion"), opts.MinVersion,
				fmt.Sprintf("must be one of: %s, %s, %s, %s",
					configapi.TLSVersion10, configapi.TLSVersion11, configapi.TLSVersion12, configapi.TLSVersion13)))
		}
	}

	for i, name := range opts.CipherSuites {
		if isRC4Suite(name) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("cipherSuites").Index(i), name,
				"RC4 cipher suites are prohibited by RFC 7465"))
			continue
		}
		if _, recognized, _ := lookupCipherSuiteID(name); !recognized {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("cipherSuites").Index(i), name,
				"must be a Go crypto/tls cipher suite name (see https://pkg.go.dev/crypto/tls#CipherSuites)"))
		}
	}

	for i, name := range opts.CurvePreferences {
		if _, ok := supportedCurves[name]; !ok {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("curvePreferences").Index(i), name,
				fmt.Sprintf("must be one of: %s", strings.Join(supportedCurveNames(), ", "))))
		}
	}

	return allErrs
}

// supportedCurveNames returns the recognized curve preference names, sorted,
// for inclusion in validation errors.
func supportedCurveNames() []string {
	names := make([]string, 0, len(supportedCurves))
	for name := range supportedCurves {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

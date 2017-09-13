package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/ghodss/yaml"

	"github.com/deepinsight/mlkube.io/pkg/controller"
	"github.com/deepinsight/mlkube.io/pkg/garbagecollection"
	"github.com/deepinsight/mlkube.io/pkg/util"
	"github.com/deepinsight/mlkube.io/pkg/util/k8sutil"
	"github.com/deepinsight/mlkube.io/pkg/util/k8sutil/election"
	"github.com/deepinsight/mlkube.io/pkg/util/k8sutil/election/resourcelock"
	"github.com/deepinsight/mlkube.io/version"

	log "github.com/golang/glog"

	"io/ioutil"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
	"github.com/deepinsight/mlkube.io/pkg/spec"
)

var (
	namespace  string
	name       string
	gcInterval time.Duration

	chaosLevel           int
	controllerConfigFile string
	printVersion         bool
)

var (
	controllerConfig = spec.ControllerConfig{}
	leaseDuration    = 15 * time.Second
	renewDuration    = 5 * time.Second
	retryPeriod      = 3 * time.Second
	tfJobClient      *k8sutil.TfJobRestClient
)

func init() {
	// chaos level will be removed once we have a formal tool to inject failures.
	flag.IntVar(&chaosLevel, "chaos-level", -1, "DO NOT USE IN PRODUCTION - level of chaos injected into the TfJob created by the operator.")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.DurationVar(&gcInterval, "gc-interval", 10*time.Minute, "GC interval")
	flag.StringVar(&controllerConfigFile, "controller_config_file", "", "Path to file containing the controller config.")

	flag.Parse()

	// Workaround for watching TPR resource.
	restCfg, err := k8sutil.InClusterConfig()
	if err != nil {
		panic(err)
	}
	controller.MasterHost = restCfg.Host
	tfJobClient, err = k8sutil.NewTfJobClient()
	if err != nil {
		panic(err)
	}
	controller.KubeHttpCli = tfJobClient.Client()

	if controllerConfigFile != "" {
		log.Infof("Loading controller config from %v.", controllerConfigFile)

		data, err := ioutil.ReadFile(controllerConfigFile)
		if err != nil {
			log.Fatalf("Could not read file: %v. Error: %v", controllerConfigFile, err)
		}

		err = yaml.Unmarshal(data, &controllerConfig)
		if err != nil {
			log.Fatalf("Could not parse controller config; Error: %v\n", err)
		}

		log.Infof("ControllerConfig: %v", util.Pformat(controllerConfig))
	} else {
		log.Info("No controller_config_file provided; using empty config.")
	}
}

func main() {
	namespace = os.Getenv("MY_POD_NAMESPACE")
	if len(namespace) == 0 {
		log.Fatalf("must set env MY_POD_NAMESPACE")
	}
	name = os.Getenv("MY_POD_NAME")
	if len(name) == 0 {
		log.Fatalf("must set env MY_POD_NAME")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c)
	go func() {
		log.Infof("received signal: %v", <-c)
		os.Exit(1)
	}()

	if printVersion {
		fmt.Println("tf_operator Version:", version.Version)
		fmt.Println("Git SHA:", version.GitSHA)
		fmt.Println("Go Version:", runtime.Version())
		fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	log.Infof("tf_operator Version: %v", version.Version)
	log.Infof("Git SHA: %s", version.GitSHA)
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	id, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	// TODO: replace with to client-go once leader election pacakge is imported
	// see https://github.com/kubernetes/client-go/issues/28
	rl := &resourcelock.EndpointsLock{
		EndpointsMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      "tf-operator",
		},
		Client: k8sutil.MustNewKubeClient(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: &record.FakeRecorder{},
		},
	}

	election.RunOrDie(election.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				log.Fatalf("leader election lost")
			},
		},
	})

	panic("unreachable")
}

func run(stop <-chan struct{}) {
	kubeCli := k8sutil.MustNewKubeClient()

	go periodicFullGC(kubeCli, tfJobClient, namespace, gcInterval)

	// TODO(jlewi): Should we start chaos?
	// startChaos(context.Background(), cfg.KubeCli, cfg.Namespace, chaosLevel)
	apiCli := k8sutil.MustNewApiExtensionsClient()

	for {
		c := controller.New(kubeCli, apiCli, tfJobClient, namespace, controllerConfig)
		err := c.Run()
		switch err {
		case controller.ErrVersionOutdated:
		default:
			log.Fatalf("controller Run() ended with failure: %v", err)
		}
	}
}

func periodicFullGC(kubecli kubernetes.Interface, tfJobClient k8sutil.TfJobClient, ns string, d time.Duration) {
	gc := garbagecollection.New(kubecli, tfJobClient, ns)
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		<-timer.C
		err := gc.FullyCollect()
		if err != nil {
			log.Warningf("failed to cleanup resources: %v", err)
		}
	}
}

// TODO(jlewi): We should add chaos back.
//func startChaos(ctx context.Context, kubecli kubernetes.Interface, ns string, chaosLevel int) {
//  m := chaos.NewMonkeys(kubecli)
//  ls := labels.SelectorFromSet(map[string]string{"app": "etcd"})
//
//  switch chaosLevel {
//  case 1:
//    log.Info("chaos level = 1: randomly kill one etcd pod every 30 seconds at 50%")
//    c := &chaos.CrashConfig{
//      Namespace: ns,
//      Selector:  ls,
//
//      KillRate:        rate.Every(30 * time.Second),
//      KillProbability: 0.5,
//      KillMax:         1,
//    }
//    go func() {
//      time.Sleep(60 * time.Second) // don't start until quorum up
//      m.CrushPods(ctx, c)
//    }()
//
//  case 2:
//    log.Info("chaos level = 2: randomly kill at most five etcd pods every 30 seconds at 50%")
//    c := &chaos.CrashConfig{
//      Namespace: ns,
//      Selector:  ls,
//
//      KillRate:        rate.Every(30 * time.Second),
//      KillProbability: 0.5,
//      KillMax:         5,
//    }
//
//    go m.CrushPods(ctx, c)
//
//  default:
//  }
//}

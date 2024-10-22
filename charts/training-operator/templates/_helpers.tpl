{{/*
Expand the name of the chart.
*/}}
{{- define "training-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "training-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "training-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "training-operator.labels" -}}
helm.sh/chart: {{ include "training-operator.chart" . }}
{{ include "training-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "training-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "training-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service to use
*/}}
{{- define "training-operator.secretName" -}}
{{- include "training-operator.fullname" . }}-webhook-cert
{{- end }}

{{/*
Create the name of the service to use
*/}}
{{- define "training-operator.serviceName" -}}
{{- include "training-operator.fullname" . }}
{{- end }}

{{/*
Create the name of the deployment to use
*/}}
{{- define "training-operator.deploymentName" -}}
{{- include "training-operator.fullname" . }}
{{- end }}

{{/*
Create the name of the deployment to use
*/}}
{{- define "training-operator.image" -}}
{{- $imageRegistry := .Values.image.registry | default "docker.io" }}
{{- $imageRepository := .Values.image.repository | default "kubeflow/training-operator" }}
{{- $imageTag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s/%s:%s" $imageRegistry $imageRepository $imageTag }}
{{- end }}

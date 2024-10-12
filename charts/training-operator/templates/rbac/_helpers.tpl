{{/*
Create the name of the service account to use
*/}}
{{- define "training-operator.serviceAccountName" -}}
{{- include "training-operator.fullname" . }}
{{- end }}

{{/*
Create the name of the cluster role to use
*/}}
{{- define "training-operator.clusterRoleName" -}}
{{- include "training-operator.fullname" . }}
{{- end }}

{{/*
Create the name of the cluster role binding to use
*/}}
{{- define "training-operator.clusterRoleBindingName" -}}
{{- include "training-operator.fullname" . }}
{{- end }}

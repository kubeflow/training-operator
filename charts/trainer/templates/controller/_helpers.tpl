{{/*
Copyright 2024 The Kubeflow authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/}}

{{/*
Create the name of the controller.
*/}}
{{- define "trainer.controller.name" -}}
{{ include "trainer.fullname" . }}-controller
{{- end -}}

{{/*
Common labels for the controller.
*/}}
{{- define "trainer.controller.labels" -}}
{{ include "trainer.labels" . }}
app.kubernetes.io/part-of: kubeflow
app.kubernetes.io/component: controller
{{- with .Values.controller.labels }}
{{- toYaml . | nindent 0 }}
{{- end }}
{{- end -}}

{{/*
Selector labels for the controller.
*/}}
{{- define "trainer.controller.selectorLabels" -}}
{{ include "trainer.selectorLabels" . }}
app.kubernetes.io/part-of: kubeflow
app.kubernetes.io/component: controller
{{- with .Values.controller.labels }}
{{- toYaml . | nindent 0 }}
{{- end }}
{{- end -}}

{{/*
Create the name of the controller service account.
*/}}
{{- define "trainer.controller.serviceAccountName" -}}
{{ include "trainer.controller.name" . }}
{{- end -}}

{{/*
Create the name of the controller cluster role.
*/}}
{{- define "trainer.controller.clusterRoleName" -}}
{{ include "trainer.controller.name" . }}
{{- end -}}

{{/*
Create the name of the controller cluster role binding.
*/}}
{{- define "trainer.controller.clusterRoleBindingName" -}}
{{ include "trainer.controller.name" . }}
{{- end -}}

{{/*
Create the name of the controller role.
*/}}
{{- define "trainer.controller.roleName" -}}
{{ include "trainer.controller.name" . }}
{{- end -}}

{{/*
Create the name of the controller role binding.
*/}}
{{- define "trainer.controller.roleBindingName" -}}
{{ include "trainer.controller.name" . }}
{{- end -}}

{{/*
Create the name of the controller deployment.
*/}}
{{- define "trainer.controller.deploymentName" -}}
{{ include "trainer.controller.name" . }}
{{- end -}}

{{- define "trainer.controller.serviceName" -}}
{{ include "trainer.controller.name" . }}-service
{{- end -}}

{{/*
Create the name of the webhook.
*/}}
{{- define "trainer.webhook.name" -}}
{{ include "trainer.name" . }}-webhook
{{- end -}}

{{/*
Create the name of the webhook secret.
*/}}
{{- define "trainer.webhook.secretName" -}}
{{ include "trainer.webhook.name" . }}-cert
{{- end -}}

{{/*
Create the name of the validating webhook configuration.
*/}}
{{- define "trainer.validatingWebhookConfigurationName" -}}
validator.trainer.kubeflow.org
{{- end -}}

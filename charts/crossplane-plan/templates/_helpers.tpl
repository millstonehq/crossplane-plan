{{/*
Expand the name of the chart.
*/}}
{{- define "crossplane-plan.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "crossplane-plan.fullname" -}}
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
{{- define "crossplane-plan.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "crossplane-plan.labels" -}}
helm.sh/chart: {{ include "crossplane-plan.chart" . }}
{{ include "crossplane-plan.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: pr-preview
app.kubernetes.io/part-of: crossplane-platform
{{- with .Values.additionalLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "crossplane-plan.selectorLabels" -}}
app.kubernetes.io/name: {{ include "crossplane-plan.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: crossplane-plan
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "crossplane-plan.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "crossplane-plan.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
crossplane-plan image
*/}}
{{- define "crossplane-plan.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
kubedock image
*/}}
{{- define "crossplane-plan.kubedockImage" -}}
{{- printf "%s:%s" .Values.kubedock.image.repository .Values.kubedock.image.tag }}
{{- end }}

{{/*
Common annotations (including ArgoCD sync wave if specified)
*/}}
{{- define "crossplane-plan.annotations" -}}
{{- if .Values.syncWave }}
argocd.argoproj.io/sync-wave: {{ .Values.syncWave | quote }}
{{- end }}
{{- with .Values.additionalAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
ConfigMap name
*/}}
{{- define "crossplane-plan.configMapName" -}}
{{- printf "%s-config" (include "crossplane-plan.fullname" .) }}
{{- end }}

{{- define "weather-exporter.name" -}}
weather-exporter
{{- end }}

{{- define "weather-exporter.fullname" -}}
{{ .Release.Name }}-weather-exporter
{{- end }}

{{- define "weather-exporter.labels" -}}
app.kubernetes.io/name: {{ include "weather-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

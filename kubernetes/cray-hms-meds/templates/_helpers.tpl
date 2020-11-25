{{/*
Add helper methods here for your chart
*/}}

{{- define "cray-hms-meds.image-prefix" -}}
    {{ $base := index . "cray-service" }}
    {{- if $base.imagesHost -}}
        {{- printf "%s/" $base.imagesHost -}}
    {{- else -}}
        {{- printf "" -}}
    {{- end -}}
{{- end -}}

{{- define "cray-hms-meds.imageTag" -}}
{{- default "latest" .Chart.AppVersion -}}
{{- end -}}
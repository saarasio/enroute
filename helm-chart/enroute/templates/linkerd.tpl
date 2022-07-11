{{- if .Values.mesh.linkerD }}
piVersion: enroute.saaras.io/v1
kind: GlobalConfig
metadata:
  name: {{ include "enroute.fullname" . }}-linkerd
  labels:
    {{- include "enroute.labels" . | nindent 4 }}
spec:
  name: linkerd-global-config
  type: globalconfig_globals
  config: |
        {
          "linkerd_enabled": true
        }
{{- end }}
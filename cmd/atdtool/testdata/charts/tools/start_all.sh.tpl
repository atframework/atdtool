#!/bin/bash
{{- range $group := .Values.proc_groups }}
# group {{ $group.group }}
{{- range $proc := $group.procs }}
{{- range $instance := $proc.instances }}
start {{ $proc.name }} {{ $instance.bus_addr }}
{{- end }}
{{- end }}
{{- end }}

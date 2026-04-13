kind: config
type: {{ .Values.type_name }}
type_id: {{ .Values.type_id }}
world_id: {{ .Values.world_id }}
zone_id: {{ .Values.zone_id }}
instance_id: {{ .Values.instance_id }}
bus_addr: {{ .Values.bus_addr }}
shared: {{ .Values.shared }}
service_only: {{ .Values.service_only }}
platform: {{ .Values.atdtool_running_platform }}
extra_enabled: {{ .Values.extra.enabled }}
extra_from_module: {{ .Values.extra.from_module }}

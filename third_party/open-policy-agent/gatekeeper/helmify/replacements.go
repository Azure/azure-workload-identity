package main

var replacements = map[string]string{
	`HELMSUBST_DEPLOYMENT_CONTAINER_RESOURCES: ""`: `{{- toYaml .Values.resources | nindent 10 }}`,

	`HELMSUBST_DEPLOYMENT_NODE_SELECTOR: ""`: `{{- toYaml .Values.nodeSelector | nindent 8 }}`,

	`HELMSUBST_DEPLOYMENT_REVISION_HISTORY_LIMIT: ""`: `{{- if .Values.revisionHistoryLimit }}
  revisionHistoryLimit: {{ .Values.revisionHistoryLimit }}
  {{- end }}`,

	"HELMSUBST_DEPLOYMENT_REPLICAS": `{{ .Values.replicaCount }}`,

	`HELMSUBST_DEPLOYMENT_AFFINITY: ""`: `{{- toYaml .Values.affinity | nindent 8 }}`,

	`HELMSUBST_DEPLOYMENT_TOPOLOGY_SPREAD_CONSTRAINTS: ""`: `{{- toYaml .Values.topologySpreadConstraints | nindent 8 }}`,

	`HELMSUBST_DEPLOYMENT_TOLERATIONS: ""`: `{{- toYaml .Values.tolerations | nindent 8 }}`,

	"HELMSUBST_CONFIGMAP_AZURE_ENVIRONMENT": `{{ .Values.azureEnvironment | default "AzurePublicCloud" }}`,

	"HELMSUBST_CONFIGMAP_AZURE_TENANT_ID": `{{ required "A valid .Values.azureTenantID entry required!" .Values.azureTenantID }}`,

	`HELMSUBST_SERVICE_TYPE: ""`: `{{- if .Values.service }}
  type: {{  .Values.service.type | default "ClusterIP" }}
  {{- end }}`,

	"HELMSUBST_DEPLOYMENT_METRICS_PORT": `{{ trimPrefix ":" .Values.metricsAddr }}`,

	"HELMSUBST_DEPLOYMENT_PRIORITY_CLASS_NAME": `{{ .Values.priorityClassName }}`,

	`HELMSUBST_MUTATING_WEBHOOK_ANNOTATIONS: ""`: `{{- toYaml .Values.mutatingWebhookAnnotations | nindent 4 }}`,

	`HELMSUBST_SERVICEACCOUNT_IMAGE_PULL_SECRETS: ""`: `{{- if .Values.imagePullSecrets }}
imagePullSecrets:
{{- toYaml .Values.imagePullSecrets | nindent 2 }}
{{- end }}`,

	`HELMSUBST_MUTATING_WEBHOOK_NAMESPACE_SELECTOR`: `{{- toYaml .Values.mutatingWebhookNamespaceSelector | nindent 4 }}`,

	`HELMSUBST_POD_ANNOTATIONS: ""`: `{{- toYaml .Values.podAnnotations | trim | nindent 8 }}`,

	`minAvailable: HELMSUBST_PODDISRUPTIONBUDGET_MINAVAILABLE`: `{{- if .Values.podDisruptionBudget.minAvailable }}
  minAvailable: {{ .Values.podDisruptionBudget.minAvailable }}
  {{- end }}`,

	`HELMSUBST_PODDISRUPTIONBUDGET_MAXUNAVAILABLE: ""`: `{{- if .Values.podDisruptionBudget.maxUnavailable }}
  maxUnavailable: {{ .Values.podDisruptionBudget.maxUnavailable }}
  {{- end }}`,
}

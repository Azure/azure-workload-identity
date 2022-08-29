package main

var replacements = map[string]string{
	`HELMSUBST_DEPLOYMENT_CONTAINER_RESOURCES: ""`: `{{- toYaml .Values.resources | nindent 10 }}`,

	`HELMSUBST_DEPLOYMENT_NODE_SELECTOR: ""`: `{{- toYaml .Values.nodeSelector | nindent 8 }}`,

	"HELMSUBST_DEPLOYMENT_REPLICAS": `{{ .Values.replicaCount }}`,

	`HELMSUBST_DEPLOYMENT_AFFINITY: ""`: `{{- toYaml .Values.affinity | nindent 8 }}`,

	`HELMSUBST_DEPLOYMENT_TOLERATIONS: ""`: `{{- toYaml .Values.tolerations | nindent 8 }}`,

	"HELMSUBST_CONFIGMAP_AZURE_ENVIRONMENT": `{{ .Values.azureEnvironment | default "AzurePublicCloud" }}`,

	"HELMSUBST_CONFIGMAP_AZURE_TENANT_ID": `{{ required "A valid .Values.azureTenantID entry required!" .Values.azureTenantID }}`,

	`HELMSUBST_SERVICE_TYPE: ""`: `{{- if .Values.service }}
  type: {{  .Values.service.type | default "ClusterIP" }}
  {{- end }}`,

	"HELMSUBST_DEPLOYMENT_METRICS_PORT": `{{ trimPrefix ":" .Values.metricsAddr }}`,

	"HELMSUBST_MUTATING_WEBHOOK_FAILURE_POLICY": `{{ .Values.mutatingWebhookFailurePolicy }}`,

	"HELMSUBST_DEPLOYMENT_PRIORITY_CLASS_NAME": `{{ .Values.priorityClassName }}`,

	`HELMSUBST_MUTATING_WEBHOOK_OBJECT_SELECTOR`: `{{- toYaml .Values.mutatingWebhookObjectSelector | nindent 4 }}`,

	`HELMSUBST_MUTATING_WEBHOOK_ANNOTATIONS: ""`: `{{- toYaml .Values.mutatingWebhookAnnotations | nindent 4 }}`,

	`HELMSUBST_SERVICEACCOUNT_IMAGE_PULL_SECRETS: ""`:
`{{- if .Values.imagePullSecrets }}
imagePullSecrets:
{{- toYaml .Values.imagePullSecrets | nindent 2 }}
{{- end }}`,
}

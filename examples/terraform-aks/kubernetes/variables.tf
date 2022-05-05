variable "app_name" {
  type = string
}

variable "aks_namespace" {
  type    = string
  default = "default"
}

variable "container_image" {
  type    = string
  default = "ghcr.io/azure/azure-workload-identity/msal-python"
}

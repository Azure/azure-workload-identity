variable "app_name" {
  type        = string
  description = "The root name of the application, as an alphanumeric short value. This name will be used as an affix to create the solution resources on Azure."
}

variable "location" {
  type        = string
  description = "The Azure location on which to create the resources."
  default     = "westus"
}

variable "aks_vm_size" {
  description = "Kubernetes VM size."
  type        = string
  default     = "Standard_B2s"
}

variable "aks_namespace" {
  description = "The default namespace to be set on AKS."
  type        = string
  default     = "default"
}

variable "aks_node_count" {
  description = "Number of nodes to deploy for Kubernetes"
  type        = number
  default     = 1
}

variable "tags" {
  description = "Tags to add to resources"
  type        = map(string)
  default     = { Application = "Azure Worload Identity" }
}

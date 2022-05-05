output "resource_group_name" {
  value = local.resource_group_name
}

output "aks_name" {
  value = local.aks_name
}

output "aks_get_credentials_command" {
  value = "az aks get-credentials -g ${local.resource_group_name} -n ${local.aks_name}"
}

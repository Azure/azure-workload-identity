terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.4.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "2.22.0"
    }
  }
  backend "local" {
    path = "./.workspace/terraform.tfstate"
  }
}

provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
    key_vault {
      purge_soft_delete_on_destroy = true
    }
  }
}

### Local Variables

locals {
  app_name             = var.app_name
  aks_namespace        = var.aks_namespace
  service_account_name = "workload-identity-sa"
  aks_node_count       = var.aks_node_count
  tags                 = var.tags
}


### Resource Group

resource "azurerm_resource_group" "default" {
  name     = "rg-${local.app_name}"
  location = var.location
  tags     = local.tags
}


### Kubernetes Cluster

resource "azurerm_kubernetes_cluster" "default" {
  name                = "aks-${local.app_name}"
  resource_group_name = azurerm_resource_group.default.name
  location            = azurerm_resource_group.default.location
  dns_prefix          = "aks-${local.app_name}"
  node_resource_group = "rg-k8s-${local.app_name}"

  oidc_issuer_enabled = true

  default_node_pool {
    name       = local.aks_namespace
    node_count = local.aks_node_count
    vm_size    = var.aks_vm_size
  }

  identity {
    type = "SystemAssigned"
  }

  tags = local.tags
}


### Azure Active Directory

resource "azuread_application" "default" {
  display_name = "aks-${local.app_name}-service-principal"
}

resource "azuread_service_principal" "default" {
  application_id               = azuread_application.default.application_id
  app_role_assignment_required = false
}

resource "azuread_application_federated_identity_credential" "default" {
  application_object_id = azuread_application.default.object_id
  display_name          = "kubernetes-federated-credential"
  description           = "Kubernetes service account federated credential"
  audiences             = ["api://AzureADTokenExchange"]
  issuer                = azurerm_kubernetes_cluster.default.oidc_issuer_url
  subject               = "system:serviceaccount:${local.aks_namespace}:${local.service_account_name}"
}


### Key Vault

data "azurerm_client_config" "current" {}

resource "azurerm_key_vault" "default" {
  name                       = "kv-${local.app_name}"
  resource_group_name        = azurerm_resource_group.default.name
  location                   = azurerm_resource_group.default.location
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  soft_delete_retention_days = 7
  purge_protection_enabled   = false

  sku_name = "standard"

  tags = local.tags
}

resource "azurerm_key_vault_access_policy" "superadmin" {
  key_vault_id = azurerm_key_vault.default.id

  tenant_id = data.azurerm_client_config.current.tenant_id
  object_id = data.azurerm_client_config.current.object_id

  secret_permissions = [
    "Backup",
    "Delete",
    "Get",
    "List",
    "Purge",
    "Recover",
    "Restore",
    "Set"
  ]
}

resource "azurerm_key_vault_access_policy" "aks" {
  key_vault_id = azurerm_key_vault.default.id

  tenant_id = data.azurerm_client_config.current.tenant_id
  object_id = azuread_service_principal.default.object_id

  secret_permissions = [
    "Get"
  ]
}

resource "azurerm_key_vault_secret" "default" {
  name         = "my-secret"
  value        = "Hello!"
  key_vault_id = azurerm_key_vault.default.id

  depends_on = [
    azurerm_key_vault_access_policy.superadmin
  ]
}


### Outputs

output "resource_group_name" {
  value = azurerm_resource_group.default.name
}

output "aks_name" {
  value = azurerm_kubernetes_cluster.default.name
}

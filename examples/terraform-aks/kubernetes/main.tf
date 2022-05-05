terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.11.0"
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
  }
}

### Local Variables

locals {
  resource_group_name   = "rg-${var.app_name}"
  keyvault_name         = "kv-${var.app_name}"
  aks_name              = "aks-${var.app_name}"
  app_registration_name = "aks-${var.app_name}-service-principal"
  service_account_name  = "workload-identity-sa"
}

### Connect to Kubernetes with Interpoation

data "azurerm_client_config" "current" {}

data "azurerm_key_vault" "main" {
  name                = local.keyvault_name
  resource_group_name = local.resource_group_name
}

data "azurerm_kubernetes_cluster" "main" {
  name                = local.aks_name
  resource_group_name = local.resource_group_name
}

provider "kubernetes" {
  host = data.azurerm_kubernetes_cluster.main.kube_config[0].host

  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.main.kube_config[0].client_certificate)
  client_key             = base64decode(data.azurerm_kubernetes_cluster.main.kube_config[0].client_key)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.main.kube_config[0].cluster_ca_certificate)
}


### App Registration for the Workload Identity

data "azuread_application" "default" {
  display_name = local.app_registration_name
}

resource "kubernetes_service_account" "default" {
  metadata {
    name      = local.service_account_name
    namespace = var.aks_namespace
    annotations = {
      "azure.workload.identity/client-id" = data.azuread_application.default.application_id
    }
    labels = {
      "azure.workload.identity/use" : "true"
    }
  }
}

### Deploy the Pod to Kubernetes

resource "kubernetes_pod" "quick_start" {
  metadata {
    name      = "quick-start"
    namespace = var.aks_namespace
  }

  spec {
    service_account_name = local.service_account_name
    container {
      image = var.container_image
      name  = "oidc"

      env {
        name  = "KEYVAULT_NAME"
        value = local.keyvault_name
      }

      env {
        name  = "SECRET_NAME"
        value = "my-secret"
      }
    }
    node_selector = {
      "kubernetes.io/os" : "linux"
    }
  }

  depends_on = [
    kubernetes_service_account.default
  ]

  lifecycle {
    ignore_changes = [
      spec[0].container[0].env["AZURE_CLIENT_ID"],
      spec[0].container[0].env["AZURE_TENANT_ID"],
      spec[0].container[0].env["AZURE_FEDERATED_TOKEN_FILE"],
      spec[0].container[0].env["AZURE_AUTHORITY_HOST"],
      spec[0].container[0].volume_mount
    ]
  }
}

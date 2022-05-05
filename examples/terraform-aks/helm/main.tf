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
    helm = {
      source  = "hashicorp/helm"
      version = "2.5.1"
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

locals {
  aks_name            = "aks-${var.app_name}"
  resource_group_name = "rg-${var.app_name}"
}

data "azurerm_kubernetes_cluster" "default" {
  name                = local.aks_name
  resource_group_name = local.resource_group_name
}

provider "helm" {
  kubernetes {
    host = data.azurerm_kubernetes_cluster.default.kube_config[0].host

    client_certificate     = base64decode(data.azurerm_kubernetes_cluster.default.kube_config[0].client_certificate)
    client_key             = base64decode(data.azurerm_kubernetes_cluster.default.kube_config[0].client_key)
    cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.default.kube_config[0].cluster_ca_certificate)
  }
}

locals {
  awi_namespace = "azure-workload-identity-system"
}


data "azurerm_client_config" "current" {}

resource "helm_release" "awi_webhook" {
  name       = "azure-workload-identity"
  chart      = "workload-identity-webhook"
  repository = "https://azure.github.io/azure-workload-identity/charts"

  namespace        = local.awi_namespace
  create_namespace = true

  set {
    name  = "azureTenantID"
    value = data.azurerm_client_config.current.tenant_id
  }
}

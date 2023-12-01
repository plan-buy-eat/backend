data "terraform_remote_state" "aks" {
  backend = "local"

  config = {
    path = "../learn-terraform-provision-aks-cluster/terraform.tfstate"
  }
}

# Retrieve AKS cluster information
provider "azurerm" {
  features {}

  client_id       = var.appId
  client_secret   = var.password
  tenant_id       = var.tenant
  subscription_id = var.subscriptionId
}

data "azurerm_kubernetes_cluster" "cluster" {
  name                = data.terraform_remote_state.aks.outputs.kubernetes_cluster_name
  resource_group_name = data.terraform_remote_state.aks.outputs.resource_group_name
}

provider "kubernetes" {
  host = data.azurerm_kubernetes_cluster.cluster.kube_config.0.host

  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config.0.client_certificate)
  client_key             = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config.0.client_key)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config.0.cluster_ca_certificate)
}

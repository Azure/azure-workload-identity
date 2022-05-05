# Azure Workload Identity w/ Terraform + AKS

Terraform modules to create an AKS Cluster with active OIDC that integrates with Workload Identity, allowing your pods to connect to Azure resources using Azure AD Application.

This example is a Terraform implementation of the Workload Identity [Quick Start](https://azure.github.io/azure-workload-identity/docs/quick-start.html) guideline.

## Architecture

The overall architecture of the solution and it's main components. All components are managed by Terraform.

<img src=".docs/solution.drawio.svg" width=800>

## Project Structure

This project is composed by the following Terraform modules:

- **Azure** - Create the RG, AKS cluster w/oidc, KV, App Reg, Service Principal.
- **Helm** - Install the Azure Workload Identity System objects.
- **Kubernetes** - Create the Service Account and deploy a quick-start workload.

ℹ️ Modules are isolated for individual `apply` commands, following [this warning](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs#stacking-with-managed-kubernetes-cluster-resources) from the Kubernetes provider.

## Deployment Steps

### 1 - Enable OIDC Issuer Preview

Head over to this Microsoft Docs section: **[Register the `EnableOIDCIssuerPreview` feature flag](https://docs.microsoft.com/en-us/azure/aks/cluster-configuration#register-the-enableoidcissuerpreview-feature-flag)**

Enable the feature (`az feature register`) and propagate it (`az provider register`).

Then return here and continue. You don't need to install or create anything else as everything will be configured and managed by the Terraform modules.

### 2 - Prepare the local variables

Create the local variables from the example file:

```sh
# Copy from the template
cp .config/example.local.tfvars .local.tfvars

# Set is as relative to work from the modules root
tfvars='../.local.tfvars'
```

You might want to change the `app_name` value to avoid conflict with existing resources. Just make sure that `kv-${app_name}` won't exceed 24 characters, as this is the Key Vault limit.

All other variables are optional and have default values, but you may edit to fit your needs.

### 3 - Deploy the Resources

Create the Azure Cloud resources:

```bash
terraform -chdir='azure' init
terraform -chdir='azure' plan -var-file=$tfvars
terraform -chdir='azure' apply -var-file=$tfvars -auto-approve
```

Apply the Helm module:

```bash
terraform -chdir='helm' init
terraform -chdir='helm' plan -var-file=$tfvars
terraform -chdir='helm' apply -var-file=$tfvars -auto-approve
```

Apply the Kubernetes module:

```bash
terraform -chdir='kubernetes' init
terraform -chdir='kubernetes' plan -var-file=$tfvars
terraform -chdir='kubernetes' apply -var-file=$tfvars -auto-approve
```
On your own solutions you might choose to use `yaml` files, but here we are making it everything managed by TF for convenience.

That's it, you can now copy the output `aks_get_credentials_command` variable to test Workload Identity with the `quick-start` container.


### 4 - Test the workload

Connect using `kubectl` and check the response:

```bash
az aks get-credentials -g '<resource-group-name>' -n '<aks-name>'

kubectl logs quick-start
```

You should see the output: `successfully got secret, secret=Hello!`

---

### Clean Up

Delete the resources to avoid unwanted costs:

```bash
terraform -chdir='azure' destroy -var-file=$tfvars -auto-approve
```

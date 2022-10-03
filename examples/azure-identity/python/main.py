import os

from azure.keyvault.secrets import SecretClient
from azure.identity import DefaultAzureCredential

def main():
    keyvault_url = os.getenv('KEYVAULT_URL', '')
    secret_name = os.getenv('SECRET_NAME', '')

    # create a secret client with the default credential
    # DefaultAzureCredential will use the environment variables injected by the Azure Workload Identity
    # mutating webhook to authenticate with Azure Key Vault.
    keyvault_client = SecretClient(vault_url=keyvault_url, credential=DefaultAzureCredential())
    secret = keyvault_client.get_secret(secret_name)
    print('successfully got secret, secret={}'.format(secret.value))

if __name__ == '__main__':
    main()

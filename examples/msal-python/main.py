import os
import time

from azure.keyvault.secrets import SecretClient
from token_credential import MyClientAssertionCredential

def main():
    # get enviornment variables to authenticate to the key vault
    azure_client_id = os.getenv('AZURE_CLIENT_ID', '')
    azure_tenant_id = os.getenv('AZURE_TENANT_ID', '')
    azure_authority_host = os.getenv('AZURE_AUTHORITY_HOST', '')
    azure_federated_token_file = os.getenv('AZURE_FEDERATED_TOKEN_FILE', '')

    # create a token credential object, which has a get_token method that returns a token
    token_credential = MyClientAssertionCredential(azure_client_id, azure_tenant_id, azure_authority_host, azure_federated_token_file)

    keyvault_name = os.getenv('KEYVAULT_NAME', '')
    secret_name = os.getenv('SECRET_NAME', '')

    # create a secret client with the token credential
    keyvault = SecretClient(vault_url='https://{}.vault.azure.net'.format(keyvault_name), credential=token_credential)
    secret = keyvault.get_secret(secret_name)
    print('successfully got secret, secret={}'.format(secret.value))

if __name__ == '__main__':
    main()

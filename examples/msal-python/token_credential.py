import time

from azure.core.credentials import AccessToken
from msal import ConfidentialClientApplication

# The following code demonstrates the use of the msal library to
# authenticate with a service using client assertion.
class MyClientAssertionCredential(object):

    def __init__(self, azure_client_id, azure_tenant_id, azure_authority_host, azure_federated_token_file):
        # read the projected service account token file
        f = open(azure_federated_token_file, 'rb')
        # create a confidential client application
        self.app = ConfidentialClientApplication(
            azure_client_id,
            client_credential={
                'client_assertion': f.read().decode("utf-8")
            },
            authority="{}{}".format(azure_authority_host, azure_tenant_id)
        )

    def get_token(self, *scopes, **kwargs):
        # get the token using the application
        token = self.app.acquire_token_for_client(scopes)
        if 'error' in token:
            raise Exception(token['error_description'])
        expires_on = time.time() + token['expires_in']
        # return an access token with the token string and expiration time
        return AccessToken(token['access_token'], int(expires_on))

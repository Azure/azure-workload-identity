// <directives>
using Microsoft.Identity.Client;
using System;
using System.Threading;
using System.Collections.Generic;
using Microsoft.Azure.KeyVault;
using System.Threading.Tasks;
// <directives>

namespace akvdotnet
{
    public class Program
    {
        static void Main(string[] args)
        {
            Program P = new Program();
            string keyvaultName = Environment.GetEnvironmentVariable("KEYVAULT_NAME");
            string secretName = Environment.GetEnvironmentVariable("SECRET_NAME");

            // keyvault URL
            string keyvaultURL = "https://" + keyvaultName + ".vault.azure.net/";

            KeyVaultClient kvClient = new KeyVaultClient(new KeyVaultClient.AuthenticationCallback(P.AcquireTokenUsingMSAL));

            while (true)
            {
                Console.WriteLine($"{Environment.NewLine}START {DateTime.UtcNow} ({Environment.MachineName})");

                var fetchedSecret = P.GetSecret(kvClient, keyvaultURL, secretName);
                var secretValue = fetchedSecret.Result;
                Console.WriteLine("Your secret is " + secretValue);

                // sleep and retry periodically
                Thread.Sleep(600000);
            }
        }

        public async Task<string> GetSecret(KeyVaultClient kvClient, string kvURL, string secretName)
        {
            // <getsecret>                
            var keyvaultSecret = await kvClient.GetSecretAsync($"{kvURL}", secretName);
            return keyvaultSecret.Value;
        }

        public async Task<string> AcquireTokenUsingMSAL(string authority, string resource, string scope)
        {
            // <authentication>
            // AAD Pod Identity webhook will inject the following env vars
            // 	AZURE_CLIENT_ID with the clientID set in the service account annotation
            // 	AZURE_TENANT_ID with the tenantID set in the service account annotation. If not defined, then
            // 		the tenantID provided via aad-pi-webhook-config for the webhook will be used.
            // 	TOKEN_FILE_PATH is the service account token path
            var client_id = Environment.GetEnvironmentVariable("AZURE_CLIENT_ID");
            var token_path = Environment.GetEnvironmentVariable("TOKEN_FILE_PATH");
            var tenant_id = Environment.GetEnvironmentVariable("AZURE_TENANT_ID");

            string[] _scopes = new string[] { resource + "/.default" };

            // read the service account token from the filesystem
            string signedClientAssertion = ReadJWTFromFS(token_path);

            // TODO (aramase) remove query params after available in prod
            Dictionary<string, string> otherParams = new Dictionary<string, string>();
            otherParams.Add("dc", "ESTS-PUB-WUS2-AZ1-FD000-TEST1");
            otherParams.Add("fiextoidc", "true");

            AuthenticationResult authResult = null;
            IConfidentialClientApplication confidentialClient;
            confidentialClient = ConfidentialClientApplicationBuilder.Create(client_id)
                .WithAuthority(authority)
                .WithClientAssertion(signedClientAssertion)
                .WithTenantId(tenant_id).Build();

            try
            {
                authResult = await confidentialClient.AcquireTokenForClient(_scopes)
                                .WithExtraQueryParameters(otherParams)
                                .ExecuteAsync();
            }
            catch (MsalUiRequiredException ex)
            {
                // The application doesn't have sufficient permissions.
                // - Did you declare enough app permissions during app creation?
                // - Did the tenant admin grant permissions to the application?
            }
            catch (MsalServiceException ex) when (ex.Message.Contains("AADSTS70011"))
            {
                // Invalid scope. The scope has to be in the form "https://resourceurl/.default"
                // Mitigation: Change the scope to be as expected.
            }
            return authResult.AccessToken;
        }

        public string ReadJWTFromFS(string token_path)
        {
            string text = System.IO.File.ReadAllText(token_path);
            return text;
        }
    }
}

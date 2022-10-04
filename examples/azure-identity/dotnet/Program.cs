// <directives>
using System;
using Azure.Identity;
using Azure.Security.KeyVault.Secrets;
// <directives>

namespace akvdotnet
{
    public class Program
    {
        static void Main(string[] args)
        {
            Program P = new Program();
            string keyvaultURL = Environment.GetEnvironmentVariable("KEYVAULT_URL");
            string secretName = Environment.GetEnvironmentVariable("SECRET_NAME");

            // DefaultAzureCredential will use the environment variables injected by the Azure Workload Identity
            // mutating webhook to authenticate with Azure Key Vault.
            SecretClient client = new SecretClient(
                new Uri(keyvaultURL),
                new DefaultAzureCredential());

            // <getsecret>
            var keyvaultSecret = client.GetSecret(secretName).Value;
            Console.WriteLine("Your secret is " + keyvaultSecret.Value);
        }
    }
}

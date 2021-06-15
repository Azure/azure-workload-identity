// <directives>
using System;
using System.Threading;
using Azure.Security.KeyVault.Secrets;
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

            SecretClient client = new SecretClient(
                new Uri(keyvaultURL),
                new MyClientAssertionCredential());

            while (true)
            {
                Console.WriteLine($"{Environment.NewLine}START {DateTime.UtcNow} ({Environment.MachineName})");

                // <getsecret>                
                var keyvaultSecret = client.GetSecret(secretName).Value;
                Console.WriteLine("Your secret is " + keyvaultSecret.Value);

                // sleep and retry periodically
                Thread.Sleep(600000);
            }
        }
    }
}

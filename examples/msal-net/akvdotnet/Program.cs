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
            string keyvaultURL = Environment.GetEnvironmentVariable("KEYVAULT_URL");
            if (string.IsNullOrEmpty(keyvaultURL)) {
                Console.WriteLine("KEYVAULT_URL environment variable not set");
                return;
            }

            string secretName = Environment.GetEnvironmentVariable("SECRET_NAME");
            if (string.IsNullOrEmpty(secretName)) {
                Console.WriteLine("SECRET_NAME environment variable not set");
                return;
            }

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

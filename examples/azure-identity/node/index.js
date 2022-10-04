/**
 * @summary Uses a SecretClient and DefaultAzureCredential to get a secret from a Key Vault.
 */

import { DefaultAzureCredential } from "@azure/identity";
import { SecretClient } from "@azure/keyvault-secrets";

const main = async () => {
    const keyvaultURL = process.env["KEYVAULT_URL"];
    const secretName = process.env["SECRET_NAME"];

    // DefaultAzureCredential will use the environment variables injected by the Azure Workload Identity
    // mutating webhook to authenticate with Azure Key Vault.
    const credential = new DefaultAzureCredential();
    const client = new SecretClient(keyvaultURL, credential);

    const secret = await client.getSecret(secretName);
    console.log(`successfully got secret, secret=${secret.value}`);
}

main().catch((error) => {
    console.error("An error occurred:", error);
    process.exit(1);
});

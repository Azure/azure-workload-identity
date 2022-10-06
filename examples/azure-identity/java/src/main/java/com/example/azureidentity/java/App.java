package com.example.azureidentity.java;

import java.util.Map;

import com.azure.security.keyvault.secrets.SecretClient;
import com.azure.security.keyvault.secrets.SecretClientBuilder;
import com.azure.security.keyvault.secrets.models.KeyVaultSecret;
import com.azure.identity.DefaultAzureCredentialBuilder;
import com.azure.identity.DefaultAzureCredential;

public class App {
    public static void main(String[] args) {
        Map<String, String> env = System.getenv();
        String keyvaultURL = env.get("KEYVAULT_URL");
        String secretName = env.get("SECRET_NAME");

        DefaultAzureCredential defaultCredential = new DefaultAzureCredentialBuilder().build();

        SecretClient secretClient = new SecretClientBuilder()
                .vaultUrl(keyvaultURL)
                .credential(defaultCredential)
                .buildClient();
        KeyVaultSecret secret = secretClient.getSecret(secretName);
        System.out.printf("successfully got secret, secret=%s", secret.getValue());
    }
}

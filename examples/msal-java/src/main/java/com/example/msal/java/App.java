package com.example.msal.java;

import java.util.Map;

import com.azure.security.keyvault.secrets.SecretClient;
import com.azure.security.keyvault.secrets.SecretClientBuilder;
import com.azure.security.keyvault.secrets.models.KeyVaultSecret;

public class App {
    public static void main(String[] args) {
        Map<String, String> env = System.getenv();
        String keyvaultURL = env.get("KEYVAULT_URL");
        if (keyvaultURL == null) {
            System.out.println("KEYVAULT_URL environment variable not set");
            return;
        }
        String secretName = env.get("SECRET_NAME");
        if (secretName == null) {
            System.out.println("SECRET_NAME environment variable not set");
            return;
        }

        SecretClient secretClient = new SecretClientBuilder()
                .vaultUrl(keyvaultURL)
                .credential(new CustomTokenCredential())
                .buildClient();
        KeyVaultSecret secret = secretClient.getSecret(secretName);
        System.out.printf("successfully got secret, secret=%s", secret.getValue());
    }
}

package com.example.msal.java;

import java.util.Map;

import com.azure.security.keyvault.secrets.SecretClient;
import com.azure.security.keyvault.secrets.SecretClientBuilder;
import com.azure.security.keyvault.secrets.models.KeyVaultSecret;

public class App {
    public static void main(String[] args) {
        Map<String, String> env = System.getenv();
        String keyvaultName = env.get("KEYVAULT_NAME");
        String secretName = env.get("SECRET_NAME");

        SecretClient secretClient = new SecretClientBuilder()
                .vaultUrl(String.format("https://%s.vault.azure.net", keyvaultName))
                .credential(new CustomTokenCredential())
                .buildClient();
        KeyVaultSecret secret = secretClient.getSecret(secretName);
        System.out.printf("successfully got secret, secret=%s", secret.getValue());
    }
}

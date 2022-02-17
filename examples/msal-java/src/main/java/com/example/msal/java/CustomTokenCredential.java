package com.example.msal.java;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.time.ZoneOffset;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

import com.azure.core.credential.AccessToken;
import com.azure.core.credential.TokenCredential;
import com.azure.core.credential.TokenRequestContext;
import com.microsoft.aad.msal4j.ClientCredentialFactory;
import com.microsoft.aad.msal4j.ClientCredentialParameters;
import com.microsoft.aad.msal4j.ConfidentialClientApplication;
import com.microsoft.aad.msal4j.IAuthenticationResult;
import com.microsoft.aad.msal4j.IClientCredential;

import reactor.core.publisher.Mono;

public class CustomTokenCredential implements TokenCredential {
    public Mono<AccessToken> getToken(TokenRequestContext request) {
        Map<String, String> env = System.getenv();
        String clientAssertion;
        try {
            clientAssertion = new String(Files.readAllBytes(Paths.get(env.get("AZURE_FEDERATED_TOKEN_FILE"))),
                    StandardCharsets.UTF_8);
        } catch (IOException e) {
            throw new RuntimeException(e);
        }

        IClientCredential credential = ClientCredentialFactory.createFromClientAssertion(clientAssertion);
        String authority = env.get("AZURE_AUTHORITY_HOST") + env.get("AZURE_TENANT_ID");
        try {
            ConfidentialClientApplication app = ConfidentialClientApplication
                    .builder(env.get("AZURE_CLIENT_ID"), credential).authority(authority).build();

            Set<String> scopes = new HashSet<>();
            for (String scope : request.getScopes())
                scopes.add(scope);

            ClientCredentialParameters parameters = ClientCredentialParameters.builder(scopes).build();
            IAuthenticationResult result = app.acquireToken(parameters).join();
            return Mono.just(
                    new AccessToken(result.accessToken(), result.expiresOnDate().toInstant().atOffset(ZoneOffset.UTC)));
        } catch (Exception e) {
            System.out.printf("Error creating client application: %s", e.getMessage());
            System.exit(1);
        }

        return Mono.empty();
    }
}

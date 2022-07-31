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
import com.microsoft.aad.msal4j.IClientCredential;

import reactor.core.publisher.Mono;

public class CustomTokenCredential implements TokenCredential {
    private final ConfidentialClientApplication app;
    
    public CustomTokenCredential() {
        Map<String, String> env = System.getenv();
        String clientAssertion;
        try {
            clientAssertion = new String(Files.readAllBytes(Paths.get(env.get("AZURE_FEDERATED_TOKEN_FILE"))),
                    StandardCharsets.UTF_8);

            IClientCredential credential = ClientCredentialFactory.createFromClientAssertion(clientAssertion);
            String authority = env.get("AZURE_AUTHORITY_HOST") + env.get("AZURE_TENANT_ID");
            app = ConfidentialClientApplication.builder(env.get("AZURE_CLIENT_ID"), credential)
                    .authority(authority).build();
        } catch (Exception e) {
            System.out.printf("Error creating client application: %s", e.getMessage());
            throw new RuntimeException(e);
        }
    }
    
    public Mono<AccessToken> getToken(TokenRequestContext request) {
        Set<String> scopes = new HashSet<>();
        for (String scope : request.getScopes())
            scopes.add(scope);

        ClientCredentialParameters parameters = ClientCredentialParameters.builder(scopes).build();
        return Mono.defer(() -> Mono.fromFuture(app.acquireToken(parameters))).map((result) ->
                new AccessToken(result.accessToken(), result.expiresOnDate().toInstant().atOffset(ZoneOffset.UTC)));
    }
}

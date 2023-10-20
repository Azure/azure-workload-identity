// <directives>
using Azure.Core;
using System;
using Microsoft.Identity.Client;
using System.Threading;
using System.Threading.Tasks;
// <directives>

public class MyClientAssertionCredential : TokenCredential
{
    private readonly IConfidentialClientApplication _confidentialClientApp;

    public MyClientAssertionCredential()
    {
        // <authentication>
        // Azure AD Workload Identity webhook will inject the following env vars
        // 	AZURE_CLIENT_ID with the clientID set in the service account annotation
        // 	AZURE_TENANT_ID with the tenantID set in the service account annotation. If not defined, then
        // 		the tenantID provided via azure-wi-webhook-config for the webhook will be used.
        // 	AZURE_FEDERATED_TOKEN_FILE is the service account token path
        var clientID = Environment.GetEnvironmentVariable("AZURE_CLIENT_ID");
        var tokenPath = Environment.GetEnvironmentVariable("AZURE_FEDERATED_TOKEN_FILE");
        var tenantID = Environment.GetEnvironmentVariable("AZURE_TENANT_ID");

        _confidentialClientApp = ConfidentialClientApplicationBuilder
                .Create(clientID)
                .WithTenantId("https://login.microsoftonline.com/", tenantID) // for simplicity, this samples assume the app is in the public cloud
                .WithClientAssertion(() => ReadJWTFromFS(tokenPath))          // make sure that ReadJWTFromFS always returns a fresh JWT 
                .WithCacheOptions(CacheOptions.EnableSharedCacheOptions)      // make sure to cache the the AAD tokens in memory                
                .Build();
    }

    public override AccessToken GetToken(TokenRequestContext requestContext, CancellationToken cancellationToken)
    {
        return GetTokenAsync(requestContext, cancellationToken).GetAwaiter().GetResult();
    }

    public override async ValueTask<AccessToken> GetTokenAsync(TokenRequestContext requestContext, CancellationToken cancellationToken)
    {
        AuthenticationResult result = null;
        try
        {
            result = await _confidentialClientApp
                        .AcquireTokenForClient(requestContext.Scopes)
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
        return new AccessToken(result.AccessToken, result.ExpiresOn);
    }

    // This assumes the file is always updated with a fresh JWT 
    public string ReadJWTFromFS(string tokenPath)
    {
        string text = System.IO.File.ReadAllText(tokenPath);
        return text;
    }
}

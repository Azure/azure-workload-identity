// <directives>
using Azure.Core;
using System;
using Microsoft.Identity.Client;
using System.Threading;
using System.Threading.Tasks;
using System.Data;
// <directives>

public class MyClientAssertionCredential : TokenCredential
{
    private readonly IConfidentialClientApplication _confidentialClientApp;
    private DateTimeOffset _lastRead;
    private string _lastJWT = null;

    public MyClientAssertionCredential()
    {
        // <authentication>
        // Azure AD Workload Identity webhook will inject the following env vars
        // 	AZURE_CLIENT_ID with the clientID set in the service account annotation
        // 	AZURE_TENANT_ID with the tenantID set in the service account annotation. If not defined, then
        // 		the tenantID provided via azure-wi-webhook-config for the webhook will be used.
        //  AZURE_AUTHORITY_HOST is the Microsoft Entra authority host. It is https://login.microsoftonline.com" for the public cloud.
        // 	AZURE_FEDERATED_TOKEN_FILE is the service account token path
        var clientID = Environment.GetEnvironmentVariable("AZURE_CLIENT_ID");
        var tokenPath = Environment.GetEnvironmentVariable("AZURE_FEDERATED_TOKEN_FILE");
        var tenantID = Environment.GetEnvironmentVariable("AZURE_TENANT_ID");
        var host = Environment.GetEnvironmentVariable("AZURE_AUTHORITY_HOST");

        _confidentialClientApp = ConfidentialClientApplicationBuilder
                .Create(clientID)
                .WithAuthority(host, tenantID) 
                .WithClientAssertion(() => ReadJWTFromFSOrCache(tokenPath))   // ReadJWTFromFS should always return a non-expired JWT 
                .WithCacheOptions(CacheOptions.EnableSharedCacheOptions)      // cache the the AAD tokens in memory                
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

    /// <summary>
    /// Read the JWT from the file system, but only do this every few minutes to avoid heavy I/O.
    /// The JWT lifetime is anywhere from 1 to 24 hours, so we can safely cache the value for a few minutes.
    /// </summary>
    private string ReadJWTFromFSOrCache(string tokenPath)
    {
        // read only once every 5 minutes
        if (_lastJWT == null ||
            DateTimeOffset.UtcNow.Subtract(_lastRead) > TimeSpan.FromMinutes(5))
        {            
            _lastRead = DateTimeOffset.UtcNow;
            _lastJWT = System.IO.File.ReadAllText(tokenPath);
        }

        return _lastJWT;
    }
}

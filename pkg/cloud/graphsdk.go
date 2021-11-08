package cloud

import (
	azidentity "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	a "github.com/microsoft/kiota/authentication/go/azure"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models/microsoft/graph"
	"github.com/pkg/errors"
)

func GraphClient(tenantID, clientID, clientSecret string) (*graph.Application, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credential")
	}
	auth, err := a.NewAzureIdentityAuthenticationProviderWithScopes(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create authentication provider")
	}
	adapter, err := msgraphsdk.NewGraphRequestAdapter(auth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request adapter")
	}
	client := msgraphsdk.NewGraphServiceClient(adapter)

	name := "kiota"
	appOptions := &applications.ApplicationsRequestBuilderPostOptions{
		Body: graph.NewApplication(),
	}
	appOptions.Body.SetDisplayName(&name)

	return client.Applications().Post(appOptions)
}

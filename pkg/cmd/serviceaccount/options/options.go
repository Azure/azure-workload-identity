package options

type option struct {
	Flag        string
	Description string
}

var (
	// ServiceAccountName flag sets the service account name
	ServiceAccountName = option{
		Flag:        "service-account-name",
		Description: "Name of the service account",
	}
	// ServiceAccountNamespace flag sets the service account namespace
	ServiceAccountNamespace = option{
		Flag:        "service-account-namespace",
		Description: "Namespace of the service account",
	}
	// ServiceAccountIssuerURL flag sets the service account issuer URL
	ServiceAccountIssuerURL = option{
		Flag:        "service-account-issuer-url",
		Description: "URL of the issuer",
	}
	// ServiceAccountTokenExpiration flag sets the service account token expiration
	ServiceAccountTokenExpiration = option{
		Flag:        "service-account-token-expiration",
		Description: "Expiration time of the service account token. Must be between 1 hour and 24 hours",
	}
	// AADApplicationName flag sets the AAD application name
	AADApplicationName = option{
		Flag:        "aad-application-name",
		Description: "Name of the AAD application, If not specified, the namespace, the name of the service account and the hash of the issuer URL will be used",
	}
	// AADApplicationClientID flag sets the AAD application client ID
	AADApplicationClientID = option{
		Flag:        "aad-application-client-id",
		Description: "Client ID of the AAD application. If not specified, it will be fetched using the AAD application name",
	}
	// AADApplicationObjectID flag sets the AAD application object ID
	AADApplicationObjectID = option{
		Flag:        "aad-application-object-id",
		Description: "Object ID of the AAD application. If not specified, it will be fetched using the AAD application name",
	}
	// ServicePrincipalName flag sets the service principal name
	ServicePrincipalName = option{
		Flag:        "service-principal-name",
		Description: "Name of the service principal that backs the AAD application. If this is not specified, the name of the AAD application will be used",
	}
	// ServicePrincipalObjectID flag sets the service principal object ID
	ServicePrincipalObjectID = option{
		Flag:        "service-principal-object-id",
		Description: "Object ID of the service principal that backs the AAD application. If not specified, it will be fetched using the service principal name",
	}
	// AzureScope flag sets the Azure scope
	AzureScope = option{
		Flag:        "azure-scope",
		Description: "Scope at which the role assignment or definition applies to",
	}
	// AzureRole flag sets the Azure role
	AzureRole = option{
		Flag:        "azure-role",
		Description: "Role of the AAD application (see all available roles at https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles)",
	}
	// RoleAssignmentID flag sets the Azure role assignment ID
	RoleAssignmentID = option{
		Flag:        "role-assignment-id",
		Description: "Azure role assignment ID",
	}
)

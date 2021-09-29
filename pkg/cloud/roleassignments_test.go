package cloud

import "testing"

func TestGetRoleDefinitionID(t *testing.T) {
	tests := []struct {
		name                string
		inputSubscriptionID string
		inputRoleName       string
		want                string
	}{
		{
			name:                "contributor role",
			inputSubscriptionID: "subscription-id",
			inputRoleName:       "Contributor",
			want:                "/subscriptions/subscription-id/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c",
		},
		{
			name:                "reader role",
			inputSubscriptionID: "subscription-id",
			inputRoleName:       "Reader",
			want:                "/subscriptions/subscription-id/providers/Microsoft.Authorization/roleDefinitions/acdd72a7-3385-48ef-bd42-f606fba81ae7",
		},
		{
			name:                "handle lower case for role name",
			inputSubscriptionID: "subscription-id",
			inputRoleName:       "contributor",
			want:                "/subscriptions/subscription-id/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRoleDefinitionID(tt.inputSubscriptionID, tt.inputRoleName)
			if err != nil {
				t.Errorf("getRoleDefinitionID() returned an error: %s", err)
			}
			if got != tt.want {
				t.Errorf("getRoleDefinitionID() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetRoleDefinitionIDError(t *testing.T) {
	_, err := getRoleDefinitionID("subscription-id", "invalid")
	if err == nil {
		t.Error("getRoleDefinitionID() expected an error, got nil")
	}
}

package options

import "testing"

func TestFlagIsRequiredError(t *testing.T) {
	err := FlagIsRequiredError("name")
	if err.Error() != "--name is required" {
		t.Errorf("FlagIsRequiredError() = %v, want %v", err, "--name is required")
	}
}

func TestOneOfFlagsIsRequiredError(t *testing.T) {
	tests := []struct {
		name      string
		flagNames []string
		errorMsg  string
	}{
		{
			name:      "one flag",
			flagNames: []string{"name"},
			errorMsg:  "--name is required",
		},
		{
			name:      "two flags",
			flagNames: []string{"name", "namespace"},
			errorMsg:  "--name or --namespace is required",
		},
		{
			name:      "three flags",
			flagNames: []string{"name", "namespace", "cluster"},
			errorMsg:  "--name or --namespace or --cluster is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := OneOfFlagsIsRequiredError(test.flagNames...)
			if err.Error() != test.errorMsg {
				t.Errorf("OneOfFlagsIsRequiredError() = %v, want %v", err, test.errorMsg)
			}
		})
	}
}

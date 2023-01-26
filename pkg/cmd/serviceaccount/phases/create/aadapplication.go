package phases

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/version"
)

const (
	aadApplicationPhaseName = "aad-application"
)

type aadApplicationPhase struct {
}

// NewAADApplicationPhase creates a new phase to create an AAD application
func NewAADApplicationPhase() workflow.Phase {
	p := &aadApplicationPhase{}
	return workflow.Phase{
		Name:        aadApplicationPhaseName,
		Aliases:     []string{"app"},
		Description: "Create Azure Active Directory (AAD) application and its underlying service principal",
		PreRun:      p.prerun,
		Run:         p.run,
		Flags:       []string{options.AADApplicationName.Flag},
	}
}

func (p *aadApplicationPhase) prerun(data workflow.RunData) error {
	createData, ok := data.(CreateData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if createData.AADApplicationName() == "" {
		return options.FlagIsRequiredError(options.AADApplicationName.Flag)
	}

	return nil
}

func (p *aadApplicationPhase) run(ctx context.Context, data workflow.RunData) error {
	createData := data.(CreateData)

	// Check if the application with the same name already exists
	var err error
	app, err := createData.AADApplication()
	if err != nil {
		if !cloud.IsNotFound(err) {
			return errors.Wrap(err, "failed to get AAD application")
		}

		// create the application as it doesn't exist
		app, err = createData.AzureClient().CreateApplication(ctx, createData.AADApplicationName())
		if app == nil || err != nil {
			return errors.Wrap(err, "failed to create AAD application")
		}
	}

	mlog.WithValues(
		"name", *app.GetDisplayName(),
		"clientID", *app.GetAppId(),
		"objectID", *app.GetId(),
	).WithName(aadApplicationPhaseName).Info("created an AAD application")

	// Check if the service principal with the same name already exists
	sp, err := createData.ServicePrincipal()
	if err != nil {
		if !cloud.IsNotFound(err) {
			return errors.Wrap(err, "failed to get service principal")
		}

		// create the service principal as it doesn't exist
		tags := []string{
			fmt.Sprintf("azwi version: %s, commit: %s", version.BuildVersion, version.Vcs),
		}

		sp, err = createData.AzureClient().CreateServicePrincipal(ctx, *app.GetAppId(), tags)
		if sp == nil || err != nil {
			return errors.Wrap(err, "failed to create service principal")
		}
	}

	mlog.WithValues(
		"name", *sp.GetDisplayName(),
		"clientID", *sp.GetAppId(),
		"objectID", *sp.GetId(),
	).WithName(aadApplicationPhaseName).Info("created service principal")

	return nil
}

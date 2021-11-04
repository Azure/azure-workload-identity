package phases

import (
	"context"
	"fmt"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/version"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
		Flags:       []string{"aad-application-name"},
	}
}

func (p *aadApplicationPhase) prerun(data workflow.RunData) error {
	createData, ok := data.(CreateData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if createData.AADApplicationName() == "" {
		return errors.New("--aad-application-name is required")
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

	log.WithFields(log.Fields{
		"name":     *app.DisplayName,
		"clientID": *app.AppID,
		"objectID": *app.ObjectID,
	}).Infof("[%s] created an AAD application", aadApplicationPhaseName)

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

		sp, err = createData.AzureClient().CreateServicePrincipal(ctx, *app.AppID, tags)
		if sp == nil || err != nil {
			return errors.Wrap(err, "failed to create service principal")
		}
	}

	log.WithFields(log.Fields{
		"name":     *sp.DisplayName,
		"clientID": *sp.AppID,
		"objectID": *sp.ObjectID,
	}).Infof("[%s] created service principal", aadApplicationPhaseName)

	return nil
}

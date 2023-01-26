package phases

import (
	"context"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"monis.app/mlog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"
)

const (
	serviceAccountPhaseName = "service-account"
)

type serviceAccountPhase struct {
	kubeClient client.Client
}

// NewServiceAccountPhase creates a new phase to delete the Kubernetes service account
func NewServiceAccountPhase() workflow.Phase {
	p := &serviceAccountPhase{}
	return workflow.Phase{
		Name:        serviceAccountPhaseName,
		Aliases:     []string{"sa"},
		Description: "Delete the Kubernetes service account in the current KUBECONFIG context",
		PreRun:      p.prerun,
		Run:         p.run,
		Flags: []string{
			options.ServiceAccountNamespace.Flag,
			options.ServiceAccountName.Flag,
		},
	}
}

func (p *serviceAccountPhase) prerun(data workflow.RunData) error {
	deleteData, ok := data.(DeleteData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if deleteData.ServiceAccountNamespace() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountNamespace.Flag)
	}
	if deleteData.ServiceAccountName() == "" {
		return options.FlagIsRequiredError(options.ServiceAccountName.Flag)
	}

	var err error
	if p.kubeClient, err = deleteData.KubeClient(); err != nil {
		return errors.Wrap(err, "failed to get Kubernetes client")
	}

	return nil
}

func (p *serviceAccountPhase) run(ctx context.Context, data workflow.RunData) error {
	deleteData := data.(DeleteData)

	l := mlog.WithValues(
		"namespace", deleteData.ServiceAccountNamespace(),
		"name", deleteData.ServiceAccountName(),
	).WithName(serviceAccountPhaseName)
	err := kuberneteshelper.DeleteServiceAccount(
		ctx,
		p.kubeClient,
		deleteData.ServiceAccountNamespace(),
		deleteData.ServiceAccountName(),
	)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to delete service account")
		}
		l.Warning("service account not found")
	} else {
		l.Info("deleted service account")
	}

	return nil
}

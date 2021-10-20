package phases

import (
	"context"

	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	serviceAccountPhaseName = "service-account"
)

type serviceAccountPhase struct {
	kubeClient kubernetes.Interface
}

// NewServiceAccountPhase creates a new phase to create a Kubernetes service account
func NewServiceAccountPhase() workflow.Phase {
	p := &serviceAccountPhase{}
	return workflow.Phase{
		Name:        serviceAccountPhaseName,
		Description: "Delete Kubernetes service account in the current KUBECONFIG context and add azure-workload-identity labels and annotations to it",
		PreRun:      p.prerun,
		Run:         p.run,
	}
}

func (p *serviceAccountPhase) prerun(data workflow.RunData) error {
	deleteData, ok := data.(DeleteData)
	if !ok {
		return errors.Errorf("invalid data type %T", data)
	}

	if deleteData.ServiceAccountNamespace() == "" {
		return errors.New("--service-account-namespace is required")
	}
	if deleteData.ServiceAccountName() == "" {
		return errors.New("--service-account-name is required")
	}

	var err error
	if p.kubeClient, err = deleteData.KubeClient(); err != nil {
		return errors.Wrap(err, "failed to get Kubernetes client")
	}

	return nil
}

func (p *serviceAccountPhase) run(ctx context.Context, data workflow.RunData) error {
	createData := data.(DeleteData)

	l := log.WithFields(log.Fields{
		"namespace": createData.ServiceAccountNamespace(),
		"name":      createData.ServiceAccountName(),
	})
	err := kuberneteshelper.DeleteServiceAccount(
		ctx,
		p.kubeClient,
		createData.ServiceAccountNamespace(),
		createData.ServiceAccountName(),
	)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to delete service account")
		}
		l.Warnf("[%s] service account not found", serviceAccountPhaseName)
	} else {
		l.Infof("[%s] deleted service account", serviceAccountPhaseName)
	}

	return nil
}

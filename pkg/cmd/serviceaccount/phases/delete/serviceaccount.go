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

// NewServiceAccountPhase creates a new phase to delete the Kubernetes service account
func NewServiceAccountPhase() workflow.Phase {
	p := &serviceAccountPhase{}
	return workflow.Phase{
		Name:        serviceAccountPhaseName,
		Aliases:     []string{"sa"},
		Description: "Delete the Kubernetes service account in the current KUBECONFIG context",
		PreRun:      p.prerun,
		Run:         p.run,
		Flags:       []string{"service-account-namespace", "service-account-name"},
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
	deleteData := data.(DeleteData)

	l := log.WithFields(log.Fields{
		"namespace": deleteData.ServiceAccountNamespace(),
		"name":      deleteData.ServiceAccountName(),
	})
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
		l.Warnf("[%s] service account not found", serviceAccountPhaseName)
	} else {
		l.Infof("[%s] deleted service account", serviceAccountPhaseName)
	}

	return nil
}

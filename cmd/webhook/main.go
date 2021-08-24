package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/Azure/azure-workload-identity/pkg/util"
	"github.com/Azure/azure-workload-identity/pkg/version"
	wh "github.com/Azure/azure-workload-identity/pkg/webhook"
)

var webhooks = []rotator.WebhookInfo{
	{
		Name: "azure-wi-webhook-mutating-webhook-configuration",
		Type: rotator.Mutating,
	},
}

const (
	secretName     = "azure-wi-webhook-server-cert" // #nosec
	serviceName    = "azure-wi-webhook-webhook-service"
	caName         = "azure-workload-identity-ca"
	caOrganization = "azure-workload-identity"
)

var (
	arcCluster          bool
	audience            string
	webhookCertDir      string
	tlsMinVersion       string
	healthAddr          string
	disableCertRotation bool

	// DNSName is <service name>.<namespace>.svc
	dnsName = fmt.Sprintf("%s.%s.svc", serviceName, util.GetNamespace())
	scheme  = runtime.NewScheme()

	entryLog = log.Log.WithName("entrypoint")
)

func init() {
	log.SetLogger(zap.New())

	_ = clientgoscheme.AddToScheme(scheme)
}

func main() {
	// TODO (aramase) once webhook is added as an arc extension, use extension
	// util to check if running in arc cluster.
	flag.BoolVar(&arcCluster, "arc-cluster", false, "Running on arc cluster")
	flag.StringVar(&audience, "audience", "", "Audience for service account token")
	flag.StringVar(&webhookCertDir, "webhook-cert-dir", "/certs", "Webhook certificates dir to use. Defaults to /certs")
	flag.BoolVar(&disableCertRotation, "disable-cert-rotation", false, "disable automatic generation and rotation of webhook TLS certificates/keys")
	flag.StringVar(&tlsMinVersion, "tls-min-version", "1.3", "Minimum TLS version")
	flag.StringVar(&healthAddr, "health-addr", ":9440", "The address the health endpoint binds to")
	flag.Parse()

	// Setup a manager
	entryLog.Info("setting up manager")
	config := ctrl.GetConfigOrDie()
	config.UserAgent = version.GetUserAgent("webhook")

	// log the user agent as it makes it easier to debug issues
	entryLog.Info(config.UserAgent)

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         false,
		HealthProbeBindAddress: healthAddr,
		CertDir:                webhookCertDir,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(c)
		},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up controller manager")
		os.Exit(1)
	}

	// Make sure certs are generated and valid if cert rotation is enabled.
	setupFinished := make(chan struct{})
	if !disableCertRotation {
		entryLog.Info("setting up cert rotation")
		if err := rotator.AddRotator(mgr, &rotator.CertRotator{
			SecretKey: types.NamespacedName{
				Namespace: util.GetNamespace(),
				Name:      secretName,
			},
			CertDir:        webhookCertDir,
			CAName:         caName,
			CAOrganization: caOrganization,
			DNSName:        dnsName,
			IsReady:        setupFinished,
			Webhooks:       webhooks,
		}); err != nil {
			entryLog.Error(err, "unable to set up cert rotation")
			os.Exit(1)
		}
	} else {
		close(setupFinished)
	}

	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		entryLog.Error(err, "unable to create ready check")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		entryLog.Error(err, "unable to create health check")
		os.Exit(1)
	}

	go setupWebhook(mgr, setupFinished)

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}

func setupWebhook(mgr manager.Manager, setupFinished chan struct{}) {
	// Block until the setup (certificate generation) finishes.
	<-setupFinished

	// setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.TLSMinVersion = tlsMinVersion

	entryLog.Info("registering webhook to the webhook server")
	podMutator, err := wh.NewPodMutator(mgr.GetClient(), mgr.GetAPIReader(), arcCluster, audience)
	if err != nil {
		entryLog.Error(err, "unable to set up pod mutator")
		os.Exit(1)
	}
	hookServer.Register("/mutate-v1-pod", &webhook.Admission{Handler: podMutator})
}

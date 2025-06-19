package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"monis.app/mlog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/Azure/azure-workload-identity/pkg/metrics"
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
	audience            string
	webhookCertDir      string
	tlsMinVersion       string
	healthAddr          string
	metricsAddr         string
	disableCertRotation bool
	metricsBackend      string
	logLevel            string
	versionInfo         bool

	// DNSName is <service name>.<namespace>.svc
	dnsName = fmt.Sprintf("%s.%s.svc", serviceName, util.GetNamespace())
	scheme  = runtime.NewScheme()

	entryLog = mlog.New().WithName("entrypoint")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

func main() {
	if err := mainErr(); err != nil {
		mlog.Fatal(err)
	}
}

func mainErr() error {
	defer mlog.Setup()()

	flag.StringVar(&audience, "audience", "", "Audience for service account token")
	flag.StringVar(&webhookCertDir, "webhook-cert-dir", "/certs", "Webhook certificates dir to use. Defaults to /certs")
	flag.BoolVar(&disableCertRotation, "disable-cert-rotation", false, "disable automatic generation and rotation of webhook TLS certificates/keys")
	flag.StringVar(&tlsMinVersion, "tls-min-version", "1.3", "Minimum TLS version")
	flag.StringVar(&healthAddr, "health-addr", ":9440", "The address the health endpoint binds to")
	flag.StringVar(&metricsAddr, "metrics-addr", ":8095", "The address the metrics endpoint binds to")
	flag.StringVar(&metricsBackend, "metrics-backend", "prometheus", "Backend used for metrics")
	flag.StringVar(&logLevel, "log-level", "",
		"In order of increasing verbosity: unset (empty string), info, debug, trace and all.")
	flag.BoolVar(&versionInfo, "version", false, "Print version information and exit")
	flag.Parse()

	if versionInfo {
		return version.PrintVersionToStdout()
	}

	ctx := signals.SetupSignalHandler()

	if err := mlog.ValidateAndSetLogLevelAndFormatGlobally(ctx, mlog.LogSpec{
		Level:  mlog.LogLevel(logLevel),
		Format: mlog.FormatJSON,
	}); err != nil {
		return fmt.Errorf("invalid --log-level set: %w", err)
	}

	// nolint:staticcheck
	// controller-runtime forces use to use the deprecated logr.Logger returned by mlog.Logr here
	log.SetLogger(mlog.Logr())
	config := ctrl.GetConfigOrDie()
	config.UserAgent = version.GetUserAgent("webhook")

	// initialize metrics exporter before creating measurements
	entryLog.Info("initializing metrics backend", "backend", metricsBackend)
	if err := metrics.InitMetricsExporter(metricsBackend); err != nil {
		return fmt.Errorf("entrypoint: failed to initialize metrics exporter: %w", err)
	}

	// log the user agent as it makes it easier to debug issues
	entryLog.Info("setting up manager", "userAgent", config.UserAgent)

	tlsVersion, err := parseTLSVersion(tlsMinVersion)
	if err != nil {
		return fmt.Errorf("entrypoint: unable to parse TLS version: %w", err)
	}

	serverOpts := webhook.Options{
		CertDir: webhookCertDir,
		TLSOpts: []func(c *tls.Config){func(c *tls.Config) { c.MinVersion = tlsVersion }},
	}
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         false,
		HealthProbeBindAddress: healthAddr,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer:  webhook.NewServer(serverOpts),
		MapperProvider: apiutil.NewDynamicRESTMapper,
	})
	if err != nil {
		return fmt.Errorf("entrypoint: unable to set up controller manager: %w", err)
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
			return fmt.Errorf("entrypoint: unable to set up cert rotation: %w", err)
		}
	} else {
		close(setupFinished)
	}

	setupProbeEndpoints(mgr, setupFinished)
	go setupWebhook(mgr, setupFinished)

	entryLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("entrypoint: unable to run manager: %w", err)
	}

	return nil
}

func setupWebhook(mgr manager.Manager, setupFinished chan struct{}) {
	// Block until the setup (certificate generation) finishes.
	<-setupFinished

	hookServer := mgr.GetWebhookServer()

	// setup webhooks
	entryLog.Info("registering webhook to the webhook server")
	podMutator, err := wh.NewPodMutator(mgr.GetClient(), mgr.GetAPIReader(), audience, mgr.GetScheme(), mgr.GetConfig())
	if err != nil {
		panic(fmt.Errorf("unable to set up pod mutator: %w", err))
	}
	hookServer.Register("/mutate-v1-pod", &webhook.Admission{Handler: podMutator})
}

func setupProbeEndpoints(mgr ctrl.Manager, setupFinished chan struct{}) {
	// Block readiness on the mutating webhook being registered.
	// We can't use mgr.GetWebhookServer().StartedChecker() yet,
	// because that starts the webhook. But we also can't call AddReadyzCheck
	// after Manager.Start. So we need a custom ready check that delegates to
	// the real ready check after the cert has been injected and validator started.
	checker := func(req *http.Request) error {
		select {
		case <-setupFinished:
			return mgr.GetWebhookServer().StartedChecker()(req)
		default:
			return fmt.Errorf("certs are not ready yet")
		}
	}

	if err := mgr.AddHealthzCheck("healthz", checker); err != nil {
		panic(fmt.Errorf("unable to add healthz check: %w", err))
	}
	if err := mgr.AddReadyzCheck("readyz", checker); err != nil {
		panic(fmt.Errorf("unable to add readyz check: %w", err))
	}
	entryLog.Info("added healthz and readyz check")
}

func parseTLSVersion(tlsVersion string) (uint16, error) {
	switch tlsVersion {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("invalid TLS version. Must be one of: 1.0, 1.1, 1.2, 1.3")
	}
}

package main

import (
	"flag"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	wh "github.com/Azure/aad-pod-managed-identity/pkg/webhook"
)

var (
	arcCluster    bool
	audience      string
	tlsMinVersion string
	webhookCertDir string
	tlsMinVersion string
)

func init() {
	log.SetLogger(zap.New())
}

func main() {

	// TODO (aramase) once webhook is added as an arc extension, use extension
	// util to check if running in arc cluster.
	flag.BoolVar(&arcCluster, "arc-cluster", false, "Running on arc cluster")
	flag.StringVar(&audience, "audience", "", "Audience for service account token")
	// NOTE: {TempDir} in MacOS is created under /var/folders/ instead of /tmp
	// ref: https://github.com/kubernetes-sigs/controller-runtime/issues/900
	flag.StringVar(&webhookCertDir, "webhook-cert-dir", "", "Webhook certificates dir to use. Defaults to {TempDir}/k8s-webhook-server/serving-certs")
  flag.StringVar(&tlsMinVersion, "tls-min-version", "1.3", "Minimum TLS version")
	flag.Parse()

	entryLog := log.Log.WithName("entrypoint")

	// Setup a manager
	entryLog.Info("setting up manager")
	mgrOpts := manager.Options{
		CertDir: webhookCertDir,
	}
	mgr, err := manager.New(config.GetConfigOrDie(), mgrOpts)
	if err != nil {
		entryLog.Error(err, "unable to set up controller manager")
		os.Exit(1)
	}

	// setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.TLSMinVersion = tlsMinVersion

	entryLog.Info("registering webhook to the webhook server")
	podMutator, err := wh.NewPodMutator(mgr.GetClient(), arcCluster, audience)
	if err != nil {
		entryLog.Error(err, "unable to set up pod mutator")
		os.Exit(1)
	}
	hookServer.Register("/mutate-v1-pod", &webhook.Admission{Handler: podMutator})

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}

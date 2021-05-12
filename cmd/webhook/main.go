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

func init() {
	log.SetLogger(zap.New())
}

func main() {
	var arcCluster bool

	// TODO (aramase) once webhook is added as an arc extension, use extension
	// util to check if running in arc cluster.
	flag.BoolVar(&arcCluster, "arc-cluster", false, "Running on arc cluster")
	flag.Parse()

	entryLog := log.Log.WithName("entrypoint")

	// Setup a manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up controller manager")
		os.Exit(1)
	}

	// setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	entryLog.Info("registering webhook to the webhook server")
	podMutator, err := wh.NewPodMutator(mgr.GetClient(), arcCluster)
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

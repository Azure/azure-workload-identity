package main

import (
	"os"

	colorable "github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-workload-identity/pkg/cmd"
)

func main() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(colorable.NewColorableStdout())
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

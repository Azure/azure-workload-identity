package main

import (
	"os"

	"github.com/Azure/azure-workload-identity/pkg/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

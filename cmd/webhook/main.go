package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"

	"github.com/m0nsterrr/cert-manager-webhook-infomaniak/internal/resolver"
)

var (
	groupName = os.Getenv("GROUP_NAME")
	namespace = os.Getenv("NAMESPACE")
)

func main() {
	if groupName == "" {
		panic("GROUP_NAME must be specified")
	}

	if namespace == "" {
		panic("NAMESPACE must be specified")
	}

	cmd.RunWebhookServer(groupName, &resolver.InfomaniakDNSProviderSolver{})
}

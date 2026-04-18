package main

import (
	"log/slog"
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"

	"github.com/m0nsterrr/cert-manager-webhook-infomaniak/internal/resolver"
)

var (
	version   = "development"
	buildTime = "0"

	groupName = os.Getenv("GROUP_NAME")
	namespace = os.Getenv("NAMESPACE")
)

func main() {
	slog.Info("Starting cert-manager-webhook-infomaniak", "version", version, "build_time", buildTime)

	if groupName == "" {
		panic("GROUP_NAME must be specified")
	}

	if namespace == "" {
		panic("NAMESPACE must be specified")
	}

	cmd.RunWebhookServer(groupName, &resolver.InfomaniakDNSProviderSolver{Namespace: namespace})
}

package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/libdns/infomaniak"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultSecretName        = "cert-manager-webhook-infomaniak"
	defaultApiTokenSecretKey = "api-token"
)

var ctx = context.TODO()

// InfomaniakDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/cert-manager/cert-manager/pkg/acme/webhook.Solver`
// interface.
type InfomaniakDNSProviderSolver struct {
	Namespace string
	k8Client  kubernetes.Interface
}

type infomaniakDNSProviderConfig struct {
	SecretRef         string `json:"secretRef"`
	ApiTokenSecretKey string `json:"apiTokenSecretKey"`
}

func (c *InfomaniakDNSProviderSolver) Name() string {
	return "infomaniak"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *InfomaniakDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	dnsAPI, err := c.newDNSAPIFromK8Secret(ch)
	if err != nil {
		return fmt.Errorf("failed to create Infomaniak API client: %w", err)
	}

	zone := zoneNameFromChallenge(ch)

	record, err := getRecordFromId(*dnsAPI, ch)
	if err != nil {
		return err
	}

	_, err = dnsAPI.CreateOrUpdateRecord(ctx, zone, *record)
	return err
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *InfomaniakDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	dnsAPI, err := c.newDNSAPIFromK8Secret(ch)
	if err != nil {
		return fmt.Errorf("failed to create Infomaniak API client: %w", err)
	}

	record, err := getRecordFromId(*dnsAPI, ch)
	if err != nil {
		return err
	}

	return dnsAPI.DeleteRecord(ctx, zoneNameFromChallenge(ch), strconv.Itoa(record.ID))
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *InfomaniakDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	k8Client, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	c.k8Client = k8Client

	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (infomaniakDNSProviderConfig, error) {
	cfg := infomaniakDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func (p *InfomaniakDNSProviderSolver) newDNSAPIFromK8Secret(ch *v1alpha1.ChallengeRequest) (*infomaniak.Client, error) {
	config, err := loadConfig(ch.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if config.SecretRef == "" {
		config.SecretRef = defaultSecretName
	}

	if config.ApiTokenSecretKey == "" {
		config.ApiTokenSecretKey = defaultApiTokenSecretKey
	}

	secret, err := p.k8Client.CoreV1().Secrets(p.Namespace).Get(context.Background(), config.SecretRef, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s from namespace %s: %w", config.SecretRef, p.Namespace, err)
	}

	return &infomaniak.Client{Token: string(secret.Data[config.ApiTokenSecretKey])}, nil
}

func getRecordFromId(c infomaniak.Client, ch *v1alpha1.ChallengeRequest) (*infomaniak.IkRecord, error) {
	var record *infomaniak.IkRecord

	recordList, err := c.GetDnsRecordsForZone(ctx, zoneNameFromChallenge(ch))
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}

	for _, r := range recordList {
		if r.Source == recordNameFromChallenge(ch) && r.Target == ch.Key {
			record = &r
			break
		}
	}

	return record, nil
}

func recordNameFromChallenge(ch *v1alpha1.ChallengeRequest) string {
	return strings.TrimSuffix(ch.ResolvedFQDN, "."+ch.ResolvedZone)
}

func zoneNameFromChallenge(ch *v1alpha1.ChallengeRequest) string {
	return strings.TrimSuffix(ch.ResolvedZone, ".")
}

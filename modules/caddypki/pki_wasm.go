//go:build js && wasm

package caddypki

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule(PKI{})
}

const DefaultCAID = "local"

// PKI is a stub for Workers builds. Workers do not support local PKI,
// certificate generation, or trust-store installation.
type PKI struct {
	CAs map[string]*CA `json:"certificate_authorities,omitempty"`
}

func (PKI) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "pki",
		New: func() caddy.Module { return new(PKI) },
	}
}

func (p *PKI) Provision(caddy.Context) error { return nil }
func (p *PKI) Start() error                  { return nil }
func (p *PKI) Stop() error                   { return nil }

func (p *PKI) GetCA(caddy.Context, string) (*CA, error) {
	return nil, fmt.Errorf("pki app is unsupported in js/wasm workers builds")
}

// KeyPair is a placeholder for Caddyfile adaptation only.
type KeyPair struct {
	Certificate string `json:"certificate,omitempty"`
	PrivateKey  string `json:"private_key,omitempty"`
	Format      string `json:"format,omitempty"`
}

// CA is a placeholder for Caddyfile adaptation only.
type CA struct {
	ID                     string         `json:"id,omitempty"`
	Name                   string         `json:"name,omitempty"`
	RootCommonName         string         `json:"root_common_name,omitempty"`
	IntermediateCommonName string         `json:"intermediate_common_name,omitempty"`
	IntermediateLifetime   caddy.Duration `json:"intermediate_lifetime,omitempty"`
	MaintenanceInterval    caddy.Duration `json:"maintenance_interval,omitempty"`
	RenewalWindowRatio     float64        `json:"renewal_window_ratio,omitempty"`
	InstallTrust           *bool          `json:"install_trust,omitempty"`
	Root                   *KeyPair       `json:"root,omitempty"`
	Intermediate           *KeyPair       `json:"intermediate,omitempty"`
}

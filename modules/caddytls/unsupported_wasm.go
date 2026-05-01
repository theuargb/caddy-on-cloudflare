//go:build js && wasm

package caddytls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/certmagic"
)

func init() {
	caddy.RegisterModule(Tailscale{})
	caddy.RegisterModule(InternalIssuer{})
	caddy.RegisterModule(InlineCAPool{})
	caddy.RegisterModule(FileCAPool{})
	caddy.RegisterModule(PKIRootCAPool{})
	caddy.RegisterModule(PKIIntermediateCAPool{})
	caddy.RegisterModule(StoragePool{})
	caddy.RegisterModule(HTTPCertPool{})
	caddy.RegisterModule(SystemCAPool{})
	caddy.RegisterModule(CombinedCAPool{})
}

func unsupportedWorkersTLS(name string) error {
	return fmt.Errorf("%s is unsupported in js/wasm workers builds", name)
}

const tailscaleDomainAliasEnding = ".ts.net"

// Tailscale is unavailable in Workers builds.
type Tailscale struct{}

func (Tailscale) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "tls.get_certificate.tailscale",
		New: func() caddy.Module { return new(Tailscale) },
	}
}

func (Tailscale) Provision(caddy.Context) error {
	return unsupportedWorkersTLS("tailscale certificate manager")
}
func (Tailscale) GetCertificate(context.Context, *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return nil, unsupportedWorkersTLS("tailscale certificate manager")
}
func (Tailscale) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()
	if d.NextArg() {
		return d.ArgErr()
	}
	return nil
}

// InternalIssuer is unavailable in Workers builds.
type InternalIssuer struct {
	CA           string         `json:"ca,omitempty"`
	Lifetime     caddy.Duration `json:"lifetime,omitempty"`
	SignWithRoot bool           `json:"sign_with_root,omitempty"`
}

func (InternalIssuer) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "tls.issuance.internal",
		New: func() caddy.Module { return new(InternalIssuer) },
	}
}

func (InternalIssuer) Provision(caddy.Context) error {
	return unsupportedWorkersTLS("internal TLS issuer")
}
func (InternalIssuer) IssuerKey() string { return "internal" }
func (InternalIssuer) Issue(context.Context, *x509.CertificateRequest) (*certmagic.IssuedCertificate, error) {
	return nil, unsupportedWorkersTLS("internal TLS issuer")
}
func (InternalIssuer) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()
	for d.NextBlock(0) {
		// Accept syntax during adaptation; runtime validation rejects TLS use.
	}
	return nil
}

// CA is the interface implemented by trust pool sources.
type CA interface {
	CertPool() *x509.CertPool
}

// CertificateProvider exposes certificates for combined pools.
type CertificateProvider interface {
	Certificates() []*x509.Certificate
}

type InlineCAPool struct {
	TrustedCACerts []string `json:"trusted_ca_certs,omitempty"`
}

type FileCAPool struct {
	TrustedCACertPEMFiles   []string `json:"pem_files,omitempty"`
	TrustedCACertPEMFilesRE []string `json:"pem_files_regexp,omitempty"`
}

type PKIRootCAPool struct {
	Authority string `json:"authority,omitempty"`
}

type PKIIntermediateCAPool struct {
	Authority string `json:"authority,omitempty"`
}

type StoragePool struct {
	StorageRaw caddy.ModuleMap `json:"storage,omitempty" caddy:"namespace=caddy.storage inline_key=module"`
	Authority  string          `json:"authority,omitempty"`
}

type HTTPCertPool struct {
	Endpoints []string `json:"endpoints,omitempty"`
}

type SystemCAPool struct{}

type CombinedCAPool struct {
	SourcesRaw []caddy.ModuleMap `json:"sources,omitempty" caddy:"namespace=tls.ca_pool.source"`
	Sources    []CA              `json:"-"`
}

func (InlineCAPool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.inline", New: func() caddy.Module { return new(InlineCAPool) }}
}
func (FileCAPool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.file", New: func() caddy.Module { return new(FileCAPool) }}
}
func (PKIRootCAPool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.pki_root", New: func() caddy.Module { return new(PKIRootCAPool) }}
}
func (PKIIntermediateCAPool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.pki_intermediate", New: func() caddy.Module { return new(PKIIntermediateCAPool) }}
}
func (StoragePool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.storage", New: func() caddy.Module { return new(StoragePool) }}
}
func (HTTPCertPool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.http", New: func() caddy.Module { return new(HTTPCertPool) }}
}
func (SystemCAPool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.system", New: func() caddy.Module { return new(SystemCAPool) }}
}
func (CombinedCAPool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "tls.ca_pool.source.combine", New: func() caddy.Module { return new(CombinedCAPool) }}
}

func (InlineCAPool) Provision(caddy.Context) error  { return unsupportedWorkersTLS("inline CA pool") }
func (FileCAPool) Provision(caddy.Context) error    { return unsupportedWorkersTLS("file CA pool") }
func (PKIRootCAPool) Provision(caddy.Context) error { return unsupportedWorkersTLS("PKI root CA pool") }
func (PKIIntermediateCAPool) Provision(caddy.Context) error {
	return unsupportedWorkersTLS("PKI intermediate CA pool")
}
func (StoragePool) Provision(caddy.Context) error  { return unsupportedWorkersTLS("storage CA pool") }
func (HTTPCertPool) Provision(caddy.Context) error { return unsupportedWorkersTLS("HTTP CA pool") }
func (SystemCAPool) Provision(caddy.Context) error { return unsupportedWorkersTLS("system CA pool") }
func (CombinedCAPool) Provision(caddy.Context) error {
	return unsupportedWorkersTLS("combined CA pool")
}

func (InlineCAPool) CertPool() *x509.CertPool          { return nil }
func (FileCAPool) CertPool() *x509.CertPool            { return nil }
func (PKIRootCAPool) CertPool() *x509.CertPool         { return nil }
func (PKIIntermediateCAPool) CertPool() *x509.CertPool { return nil }
func (StoragePool) CertPool() *x509.CertPool           { return nil }
func (HTTPCertPool) CertPool() *x509.CertPool          { return nil }
func (SystemCAPool) CertPool() *x509.CertPool          { return nil }
func (CombinedCAPool) CertPool() *x509.CertPool        { return nil }

func (InlineCAPool) Certificates() []*x509.Certificate          { return nil }
func (FileCAPool) Certificates() []*x509.Certificate            { return nil }
func (PKIRootCAPool) Certificates() []*x509.Certificate         { return nil }
func (PKIIntermediateCAPool) Certificates() []*x509.Certificate { return nil }
func (StoragePool) Certificates() []*x509.Certificate           { return nil }
func (HTTPCertPool) Certificates() []*x509.Certificate          { return nil }
func (CombinedCAPool) Certificates() []*x509.Certificate        { return nil }

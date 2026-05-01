package workerd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

const defaultCaddyfile = `{
	admin off
	auto_https off
}

:80 {
	respond "caddy-on-cloudflare"
}
`

func CaddyfileFromEnv() string {
	if caddyfile := os.Getenv("CADDYFILE"); caddyfile != "" {
		return caddyfile
	}
	return defaultCaddyfile
}

func BuildServer(caddyfile string) (*caddyhttp.Server, context.CancelCauseFunc, error) {
	adapter := caddyconfig.GetAdapter("caddyfile")
	if adapter == nil {
		return nil, nil, errors.New("caddyfile adapter is not registered")
	}

	cfgJSON, warnings, err := adapter.Adapt([]byte(caddyfile), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("adapt caddyfile: %w", err)
	}
	for _, warning := range warnings {
		log.Printf("[Caddy] Caddyfile warning: %s", warning)
	}

	var cfg caddy.Config
	if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
		return nil, nil, fmt.Errorf("decode caddy config: %w", err)
	}

	ctx, cancel, err := caddy.ProvisionContextWithCancel(&cfg)
	if err != nil {
		cancel(fmt.Errorf("worker provision failed: %w", err))
		return nil, nil, fmt.Errorf("provision caddy config: %w", err)
	}

	appRaw, err := ctx.App("http")
	if err != nil {
		cancel(fmt.Errorf("worker http app failed: %w", err))
		return nil, nil, fmt.Errorf("load http app: %w", err)
	}
	app, ok := appRaw.(*caddyhttp.App)
	if !ok {
		cancel(errors.New("worker http app has unexpected type"))
		return nil, nil, fmt.Errorf("http app has unexpected type %T", appRaw)
	}

	server := selectServer(app)
	if server == nil {
		cancel(errors.New("worker config has no http servers"))
		return nil, nil, errors.New("caddy config has no http servers")
	}

	return server, cancel, nil
}

func selectServer(app *caddyhttp.App) *caddyhttp.Server {
	names := make([]string, 0, len(app.Servers))
	for name := range app.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil
	}
	return app.Servers[names[0]]
}

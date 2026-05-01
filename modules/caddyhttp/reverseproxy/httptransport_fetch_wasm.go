//go:build js && wasm

package reverseproxy

import (
	"github.com/syumai/workers/cloudflare/fetch"

	"github.com/caddyserver/caddy/v2"
)

func (h *HTTPTransport) provisionFetchTransport(caddy.Context) bool {
	h.fetchTransport = fetch.NewClient().HTTPClient(fetch.RedirectModeManual).Transport
	return true
}

//go:build !js || !wasm

package reverseproxy

import "github.com/caddyserver/caddy/v2"

func (h *HTTPTransport) provisionFetchTransport(caddy.Context) bool {
	return false
}

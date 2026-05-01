# Caddy on Cloudflare Workers

This is the first-party Workers entry point for Caddy's `js/wasm` build.

Build:

```sh
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" wasm_exec.js
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o caddy.wasm .
```

Configure Caddy with one of these, in order:

- `CADDYFILE`: inline Caddyfile content.
- `CADDYFILE_URL`: URL fetched by the JavaScript worker before Go boots.
- `ASSETS` binding with `/Caddyfile`: static Workers asset fallback.

`reverse_proxy` uses Cloudflare's `fetch()` API on `js/wasm` builds, so upstreams must be fetch-compatible HTTP or HTTPS URLs.

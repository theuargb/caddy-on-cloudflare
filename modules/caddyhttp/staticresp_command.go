//go:build !js || !wasm

package caddyhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
)

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "respond",
		Usage: `[--status <code>] [--body <content>] [--listen <addr>] [--access-log] [--debug] [--header "Field: value"] <body|status>`,
		Short: "Simple, hard-coded HTTP responses for development and testing",
		Long: `
Spins up a quick-and-clean HTTP server for development and testing purposes.

With no options specified, this command listens on a random available port
and answers HTTP requests with an empty 200 response. The listen address can
be customized with the --listen flag and will always be printed to stdout.
If the listen address includes a port range, multiple servers will be started.

If a final, unnamed argument is given, it will be treated as a status code
(same as the --status flag) if it is a 3-digit number. Otherwise, it is used
as the response body (same as the --body flag). The --status and --body flags
will always override this argument (for example, to write a body that
literally says "404" but with a status code of 200, do '--status 200 404').

A body may be given in 3 ways: a flag, a final (and unnamed) argument to
the command, or piped to stdin (if flag and argument are unset). Limited
template evaluation is supported on the body, with the following variables:

	{{.N}}        The server number (useful if using a port range)
	{{.Port}}     The listener port
	{{.Address}}  The listener address

(See the docs for the text/template package in the Go standard library for
information about using templates: https://pkg.go.dev/text/template)

Access/request logging and more verbose debug logging can also be enabled.

Response headers may be added using the --header flag for each header field.
`,
		CobraFunc: func(cmd *cobra.Command) {
			cmd.Flags().StringP("listen", "l", ":0", "The address to which to bind the listener")
			cmd.Flags().IntP("status", "s", http.StatusOK, "The response status code")
			cmd.Flags().StringP("body", "b", "", "The body of the HTTP response")
			cmd.Flags().BoolP("access-log", "", false, "Enable the access log")
			cmd.Flags().BoolP("debug", "v", false, "Enable more verbose debug-level logging")
			cmd.Flags().StringArrayP("header", "H", []string{}, "Set a header on the response (format: \"Field: value\")")
			cmd.RunE = caddycmd.WrapCommandFuncForCobra(cmdRespond)
		},
	})
}

func buildHTTPServer(
	i int,
	port uint,
	addr string,
	statusCode int,
	hdr http.Header,
	body string,
	accessLog bool,
) (*Server, error) {
	var handlers []json.RawMessage

	tplCtx := struct {
		N       int
		Port    uint
		Address string
	}{
		N:       i,
		Port:    port,
		Address: addr,
	}
	tpl, err := template.New("body").Parse(body)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = tpl.Execute(buf, tplCtx)
	if err != nil {
		return nil, err
	}

	handler := StaticResponse{
		StatusCode: WeakString(fmt.Sprintf("%d", statusCode)),
		Headers:    hdr,
		Body:       buf.String(),
	}
	handlers = append(handlers, caddyconfig.JSONModuleObject(handler, "handler", "static_response", nil))
	route := Route{HandlersRaw: handlers}

	server := &Server{
		Listen:            []string{addr},
		ReadHeaderTimeout: caddy.Duration(10 * time.Second),
		IdleTimeout:       caddy.Duration(30 * time.Second),
		MaxHeaderBytes:    1024 * 10,
		Routes:            RouteList{route},
		AutoHTTPS:         &AutoHTTPSConfig{DisableRedir: true},
	}
	if accessLog {
		server.Logs = new(ServerLogConfig)
	}

	return server, nil
}

func cmdRespond(fl caddycmd.Flags) (int, error) {
	caddy.TrapSignals()

	listen := fl.String("listen")
	statusCodeFl := fl.Int("status")
	bodyFl := fl.String("body")
	accessLog := fl.Bool("access-log")
	debug := fl.Bool("debug")
	arg := fl.Arg(0)

	if fl.NArg() > 1 {
		return caddy.ExitCodeFailedStartup, fmt.Errorf("too many unflagged arguments")
	}

	statusCode, body := statusCodeFl, bodyFl
	statusCodeFlagSpecified := slices.Contains(os.Args, "--status")

	if arg != "" {
		if bodyFl != "" && statusCodeFlagSpecified {
			return caddy.ExitCodeFailedStartup, fmt.Errorf("unflagged argument \"%s\" is overridden by flags", arg)
		}
		if argInt, err := strconv.Atoi(arg); err == nil && !statusCodeFlagSpecified {
			if argInt >= 100 && argInt <= 999 {
				statusCode = argInt
			}
		} else if body == "" {
			body = arg
		}
	}

	if body == "" {
		stdinInfo, err := os.Stdin.Stat()
		if err != nil {
			return caddy.ExitCodeFailedStartup, err
		}
		if stdinInfo.Mode()&os.ModeNamedPipe != 0 {
			bodyBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return caddy.ExitCodeFailedStartup, err
			}
			body = string(bodyBytes)
		}
	}

	headers, err := fl.GetStringArray("header")
	if err != nil {
		return caddy.ExitCodeFailedStartup, fmt.Errorf("invalid header flag: %v", err)
	}
	hdr := make(http.Header)
	for i, h := range headers {
		key, val, found := strings.Cut(h, ":")
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		if !found || key == "" || val == "" {
			return caddy.ExitCodeFailedStartup, fmt.Errorf("header %d: invalid format \"%s\" (expecting \"Field: value\")", i, h)
		}
		hdr.Set(key, val)
	}

	httpApp := App{Servers: make(map[string]*Server)}

	listenAddr, err := caddy.ParseNetworkAddress(listen)
	if err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	if !listenAddr.IsUnixNetwork() && !listenAddr.IsFdNetwork() {
		listenAddrs := make([]string, 0, listenAddr.PortRangeSize())
		for offset := uint(0); offset < listenAddr.PortRangeSize(); offset++ {
			listenAddrs = append(listenAddrs, listenAddr.JoinHostPort(offset))
		}

		for i, addr := range listenAddrs {
			server, err := buildHTTPServer(i, listenAddr.StartPort+uint(i), addr, statusCode, hdr, body, accessLog)
			if err != nil {
				return caddy.ExitCodeFailedStartup, err
			}
			httpApp.Servers[fmt.Sprintf("static%d", i)] = server
		}
	} else {
		server, err := buildHTTPServer(0, 0, listen, statusCode, hdr, body, accessLog)
		if err != nil {
			return caddy.ExitCodeFailedStartup, err
		}
		httpApp.Servers[fmt.Sprintf("static%d", 0)] = server
	}

	var false bool
	cfg := &caddy.Config{
		Admin: &caddy.AdminConfig{
			Disabled: true,
			Config: &caddy.ConfigSettings{
				Persist: &false,
			},
		},
		AppsRaw: caddy.ModuleMap{
			"http": caddyconfig.JSON(httpApp, nil),
		},
	}
	if debug {
		cfg.Logging = &caddy.Logging{
			Logs: map[string]*caddy.CustomLog{
				"default": {BaseLog: caddy.BaseLog{Level: zap.DebugLevel.CapitalString()}},
			},
		}
	}

	err = caddy.Run(cfg)
	if err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	loadedHTTPApp, err := caddy.ActiveContext().App("http")
	if err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	for _, srv := range loadedHTTPApp.(*App).Servers {
		for _, ln := range srv.listeners {
			fmt.Printf("Server address: %s\n", ln.Addr())
		}
	}

	select {}
}

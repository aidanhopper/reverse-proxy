package main

import (
	"aidanhopper/main/proxy"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
)

type basicTLS struct{}

func (c *basicTLS) Compile(conn *proxy.BufferedConn) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair("cert/server.crt", "cert/server.key")
	if err != nil {
		return nil,
			fmt.Errorf("Failed to load x509 certifcate from the filesystem with error: %s\n", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}, nil
}

// TODO:
// 1. Add router level middleware
// 2. Make service abstraction for loadbalancing, file servers, redirects, ...
// 3. Gracefully shutdown connections when an entrypoint is removed.
// 4. Gracefully shutdown connections when a router is changed
// 5.

func main() {
	server := proxy.NewServer()

	server.RegisterEntryPoint(
		&proxy.TCPEntryPoint{
			Identifer: "web-secure",
			Address:   ":443",
		},
	)

	server.RegisterEntryPoint(
		&proxy.TCPEntryPoint{
			Identifer: "web",
			Address:   ":80",
		},
	)

	server.RegisterTLSConfigCompiler(
		"web-secure",
		&basicTLS{},
	)

	compiler := proxy.NewHTTPHandleCompiler()

	compiler.
		RegisterRouter("web secure router").
		SetMiddleware(proxy.Chain(
			proxy.RequireSecure(),
			proxy.SetForwardingHeaders(),
		)).
		RegisterRoute(
			"route 1",
			&proxy.HTTPRoute{
				Rule: proxy.And(
					proxy.PathPrefix("/api/v1"),
				),
				Middleware: proxy.Chain(
					proxy.TrimPathPrefix("/api/v1/"),
					nil,
				),
				ServiceId: "cpts355 file server",
			},
		).
		RegisterRoute(
			"route 2",
			&proxy.HTTPRoute{
				Rule:       proxy.PathPrefix("/whoami"),
				Middleware: nil,
				ServiceId:  "whoami reverse proxy",
			},
		).
		RegisterRoute(
			"jellyfin reverse proxy",
			&proxy.HTTPRoute{
				Rule: proxy.And(
					proxy.PathPrefix("/jelly/"),
					proxy.Host("localhost"),
				),
				Middleware: proxy.Chain(
					proxy.TrimPathPrefix("/jelly/"),
					proxy.Logging("Jellyfin Redirect "),
				),
				ServiceId: "jellyfin reverse proxy",
			},
		).
		RegisterRoute(
			"jellyfin redirect",
			&proxy.HTTPRoute{
				Rule: proxy.PathPrefix("/jelly"),
				Middleware: proxy.Chain(
					proxy.TrimPathPrefix("/jelly"),
					proxy.Logging("Jellyfin Redirect "),
				),
				ServiceId: "jellyfin redirect",
			},
		)

	compiler.
		RegisterRouter("web router").
		RegisterRoute(
			"route 1",
			&proxy.HTTPRoute{
				Rule:      proxy.Any(),
				ServiceId: "connection upgrade to https",
			},
		)

	compiler.
		RegisterService(
			"hello web-secure",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Hello to the web entrypoint!")
			}),
		).
		RegisterService(
			"cpts355 file server",
			proxy.FileServer("../cpts355"),
		).
		RegisterService(
			"connection upgrade to https",
			proxy.UpgradeToSecure(),
		).
		RegisterService(
			"whoami reverse proxy",
			proxy.LoadBalance("http://localhost:9999"),
		).
		RegisterService(
			"jellyfin reverse proxy",
			proxy.LoadBalance(
				"http://localhost:8096",
				"http://localhost:8096",
				"http://localhost:8096",
				"http://localhost:8096",
				"http://localhost:9999",
			),
		).
		RegisterService(
			"jellyfin redirect",
			proxy.PathRedirect("/jelly/"),
		)

	server.RegisterHTTPHandler(
		"web-secure",
		compiler.Compile("web secure router"),
	)

	server.RegisterHTTPHandler(
		"web",
		compiler.Compile("web router"),
	)

	server.Serve(context.Background())
}

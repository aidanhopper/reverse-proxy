package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/aidanhopper/reverse-proxy/proxy-engine/engine"
	"github.com/aidanhopper/reverse-proxy/proxyd/proxyd"
)

func BasicTLS() engine.TLSConfigHandler {
	return engine.TLSConfigHandlerFunc(func(info *tls.ClientHelloInfo) (*tls.Config, error) {
		cert, err := tls.LoadX509KeyPair("cert/server.crt", "cert/server.key")
		if err != nil {
			return nil,
				fmt.Errorf("Failed to load x509 certifcate from the filesystem with error: %s\n", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		}, nil
	})
}

// TODO:
// 3. Gracefully shutdown connections when an entrypoint is removed.
// 4. Gracefully shutdown connections when a router is changed
// 5. Add server level filter, could check IPs as a whitelist or do rate limiting

func main() {
	server := engine.NewServer()

	server.SetFilter(nil)

	server.RegisterTLSConfigHandler(
		"web-secure",
		BasicTLS(),
	)

	server.RegisterEntryPoint(
		engine.TCPEntryPoint{
			Identifier: "web-secure",
			Address:   ":443",
		},
	)

	server.RegisterEntryPoint(
		engine.TCPEntryPoint{
			Identifier: "web",
			Address:   ":80",
		},
	)

	server.RegisterEntryPoint(
		engine.TCPEntryPoint{
			Identifier: "minecraft",
			Address:   ":25565",
		},
	)

	httpCompiler := engine.NewHTTPHandlerCompiler()
	httpCompiler.
		RegisterService(
			"whoami service",
			engine.HTTPReverseProxy("http://localhost:9999"),
		).
		RegisterService(
			"jellyfin redirect service",
			engine.PathRedirect("/jellyfin/"),
		).
		RegisterService(
			"jellyfin service",
			engine.HTTPReverseProxy("http://localhost:8096"),
		).
		RegisterService(
			"cpts355",
			engine.FileServer("../cpts355"),
		).
		RegisterRouter("router 1").
		SetMiddleware(engine.Chain(
			engine.RequireSecure(),
			engine.SetForwardingHeaders(),
		)).
		RegisterRoute(
			"whoami route",
			&engine.HTTPRoute{
				ServiceId: "whoami service",
				Rule: engine.And(
					engine.PathPrefix("/abc"),
				),
			},
		).
		RegisterRoute(
			"jellyfin route",
			&engine.HTTPRoute{
				ServiceId: "jellyfin service",
				Rule: engine.And(
					engine.PathRegexp("/jellyfin/"),
				),
				Middleware: engine.Chain(
					engine.StripPrefix("/jellyfin/"),
				),
			},
		).
		RegisterRoute(
			"jellyfin redirect route",
			&engine.HTTPRoute{
				ServiceId: "jellyfin redirect service",
				Rule: engine.And(
					engine.PathRegexp("^/jellyfin$"),
				),
			},
		).
		RegisterRoute(
			"cpts355 file server",
			&engine.HTTPRoute{
				ServiceId: "cpts355",
				Rule: engine.And(
					engine.PathPrefix("/files"),
				),
				Middleware: engine.StripPrefix("/files"),
			},
		)

	tcpCompiler := engine.NewTCPHandlerCompiler()
	tcpCompiler.
		RegisterService(
			"vanilla minecraft",
			engine.TCPReverseProxy("vanilla.mc:25565"),
		).
		RegisterRouter("router 1").
		RegisterRoute(
			"complex route",
			&engine.TCPRoute{
				ServiceId: "vanilla minecraft",
				Rule: engine.And(
					engine.HostMinecraft(
						"localhost",
					),
				),
			},
		)

	server.RegisterHTTPHandler(
		"web",
		httpCompiler.Compile("router 1"),
	)

	server.RegisterHTTPHandler(
		"web-secure",
		httpCompiler.Compile("router 1"),
	)

	server.RegisterTCPHandler(
		"minecraft",
		tcpCompiler.Compile("router 1"),
	)

	go func() {
		time.Sleep(10 * time.Second)
		server.DeregisterEntryPoint("minecraft")
	}()

	server.Serve(context.Background())
}

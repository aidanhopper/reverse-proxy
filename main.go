package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/aidanhopper/proxy/proxy"
)

func BasicTLS() proxy.TLSConfigHandler {
	return proxy.TLSConfigHandlerFunc(func(info *tls.ClientHelloInfo) (*tls.Config, error) {
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
	server := proxy.NewServer()

	server.SetFilter(nil)

	server.RegisterTLSConfigHandler(
		"web-secure",
		BasicTLS(),
	)

	server.RegisterEntryPoint(
		proxy.TCPEntryPoint{
			Identifer: "web-secure",
			Address:   ":443",
		},
	)

	server.RegisterEntryPoint(
		proxy.TCPEntryPoint{
			Identifer: "web",
			Address:   ":80",
		},
	)

	server.RegisterEntryPoint(
		proxy.TCPEntryPoint{
			Identifer: "minecraft",
			Address:   ":25565",
		},
	)

	httpCompiler := proxy.NewHTTPHandlerCompiler()
	httpCompiler.
		RegisterService(
			"whoami service",
			proxy.HTTPReverseProxy("http://localhost:9999"),
		).
		RegisterService(
			"jellyfin redirect service",
			proxy.PathRedirect("/jellyfin/"),
		).
		RegisterService(
			"jellyfin service",
			proxy.HTTPReverseProxy("http://localhost:8096"),
		).
		RegisterService(
			"cpts355",
			proxy.FileServer("../cpts355"),
		).
		RegisterRouter("router 1").
		SetMiddleware(proxy.Chain(
			proxy.RequireSecure(),
			proxy.SetForwardingHeaders(),
		)).
		RegisterRoute(
			"whoami route",
			&proxy.HTTPRoute{
				ServiceId: "whoami service",
				Rule: proxy.And(
					proxy.PathPrefix("/abc"),
				),
			},
		).
		RegisterRoute(
			"jellyfin route",
			&proxy.HTTPRoute{
				ServiceId: "jellyfin service",
				Rule: proxy.And(
					proxy.PathRegexp("/jellyfin/"),
				),
				Middleware: proxy.Chain(
					proxy.StripPrefix("/jellyfin/"),
				),
			},
		).
		RegisterRoute(
			"jellyfin redirect route",
			&proxy.HTTPRoute{
				ServiceId: "jellyfin redirect service",
				Rule: proxy.And(
					proxy.PathRegexp("^/jellyfin$"),
				),
			},
		).
		RegisterRoute(
			"cpts355 file server",
			&proxy.HTTPRoute{
				ServiceId: "cpts355",
				Rule: proxy.And(
					proxy.PathPrefix("/files"),
				),
				Middleware: proxy.StripPrefix("/files"),
			},
		)

	tcpCompiler := proxy.NewTCPHandlerCompiler()
	tcpCompiler.
		RegisterService(
			"vanilla minecraft",
			proxy.TCPReverseProxy("vanilla.mc:25565"),
		).
		RegisterRouter("router 1").
		RegisterRoute(
			"complex route",
			&proxy.TCPRoute{
				ServiceId: "vanilla minecraft",
				Rule: proxy.And(
					proxy.HostMinecraft(
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

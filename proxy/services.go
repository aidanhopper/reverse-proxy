package proxy

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

func UpgradeToSecure() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

func FileServer(dir string) http.Handler {
	return http.FileServer(http.Dir(dir))
}

func HTTPLoadBalancer(services ...http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := services[rand.Uint32()%uint32(len(services))]
		target.ServeHTTP(w, r)
	})
}

func HTTPReverseProxy(address string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetURL, err := url.Parse(address)
		if err != nil {
			log.Printf("Invalid target address provided: %s. Error: %v", address, err)
			http.NotFoundHandler()
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		originalDirector := proxy.Director // Get the default director logic

		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.RequestURI = ""
		}

		proxy.ServeHTTP(w, r)
	})
}

func Redirect(url string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url, http.StatusMovedPermanently)
	})
}

func getProtocol(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")

	if proto == "" {
		forwarded := r.Header.Get("Forwarded")
		if strings.Contains(strings.ToLower(forwarded), "proto=https") {
			return "https"
		}
	}

	if strings.ToLower(proto) == "https" {
		return "https"
	}

	return "http"
}

func PathRedirect(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := fmt.Sprintf("%s://%s%s", getProtocol(r), r.Host, path)
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

func TCPReverseProxy(address string) TCPServiceFunc {
	return TCPServiceFunc(func(conn *BufferedTCPConn) {
		remote, err := net.Dial("tcp", address)
		if err != nil {
			log.Printf("Failed to dial with error: %s\n", err)
			return
		}

		var wc sync.WaitGroup

		wc.Go(func() {
			io.Copy(conn, remote)
		})

		wc.Go(func() {
			io.Copy(remote, conn)
		})

		wc.Wait()
	})
}

func TCPLoadBalancer(services ...TCPServiceFunc) TCPServiceFunc {
	return TCPServiceFunc(func(conn *BufferedTCPConn) {
		target := services[rand.Uint32()%uint32(len(services))]
		target(conn)
	})
}

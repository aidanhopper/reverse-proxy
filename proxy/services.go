package proxy

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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

func LoadBalance(targets ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(targets) == 0 {
			log.Printf("No targets to load balance")
			http.NotFoundHandler()
		}

		target := targets[rand.Uint32()%uint32(len(targets))]

		targetURL, err := url.Parse(target)
		if err != nil {
			log.Printf("Invalid target address provided: %s. Error: %v", target, err)
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

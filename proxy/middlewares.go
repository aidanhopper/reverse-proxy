package proxy

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func Logging(prefix string) Middleware {
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		log.Printf("%s%s %s %s\n", prefix, r.Proto, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func StripPrefix(prefix string) Middleware {
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		if strings.HasPrefix(r.URL.Path, prefix) {
			r2 := r.Clone(r.Context())

			// strip prefix
			newPath := strings.TrimPrefix(r.URL.Path, prefix)
			if newPath == "" {
				newPath = "/"
			}

			r2.URL.Path = newPath
			r2.URL.RawPath = "" // let Go re-escape if needed

			next.ServeHTTP(w, r2)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireSecure() Middleware {
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		if r.TLS == nil {
			target := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SetForwardingHeaders() Middleware {
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		clientIP := r.RemoteAddr
		if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
			clientIP = clientIP[:colon]
		}

		var proto string
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}

		existingXFF := r.Header.Get("X-Forwarded-For")
		if existingXFF != "" {
			r.Header.Set("X-Forwarded-For", existingXFF+", "+clientIP)
		} else {
			r.Header.Set("X-Forwarded-For", clientIP)
		}

		r.Header.Set("X-Forwarded-Proto", proto)
		r.Header.Set("X-Forwarded-Host", r.Host)

		forwardedValue := fmt.Sprintf("for=%s; proto=%s; host=%s", clientIP, proto, r.Host)
		r.Header.Set("Forwarded", forwardedValue)

		next.ServeHTTP(w, r)
	})
}

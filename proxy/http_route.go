package proxy

import (
	"net/http"
	"strings"
)

type Rule func(*http.Request) bool

type HTTPRoute struct {
	Rule       Rule
	Middleware Middleware
	ServiceId  string
}

func Host(host string) Rule {
	return func(r *http.Request) bool {
		return r.Host == host
	}
}

func PathPrefix(prefix string) Rule {
	return func(r *http.Request) bool {
		return strings.HasPrefix(r.URL.Path, prefix)
	}
}

func And(rules ...Rule) Rule {
	return func(r *http.Request) bool {
		for _, rule := range rules {
			if !rule(r) {
				return false
			}
		}
		return true
	}
}

func Or(rules ...Rule) Rule {
	return func(r *http.Request) bool {
		for _, rule := range rules {
			if rule(r) {
				return true
			}
		}
		return false
	}
}

func Any() Rule {
	return func(_ *http.Request) bool {
		return true
	}
}

func Method(method string) Rule {
	return func(r *http.Request) bool {
		return r.Method == method
	}
}

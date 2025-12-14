package proxy

import "net/http"

type Middleware interface {
	Wrap(http.Handler) http.Handler
}

type MiddlewareFunc func(
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
)

func (f MiddlewareFunc) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f(w, r, next)
	})
}

func Chain(mws ...Middleware) Middleware {
	return MiddlewareFunc(func(
		w http.ResponseWriter,
		r *http.Request,
		next http.Handler,
	) {
		h := next
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i].Wrap(h)
		}
		h.ServeHTTP(w, r)
	})
}

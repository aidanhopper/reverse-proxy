package proxy

import "net/http"

type HTTPRouter interface {
	Match(req *http.Request) (string, *HTTPRoute)
	RegisterRoute(routeId string, route *HTTPRoute) HTTPRouter
	DeregisterRoute(routeId string)
	SetMiddleware(middleware Middleware) HTTPRouter
	Middleware() Middleware
}

type httpRouter struct {
	routes     map[string]*HTTPRoute
	middleware Middleware
}

func NewHTTPRouter() *httpRouter {
	return &httpRouter{
		make(map[string]*HTTPRoute),
		nil,
	}
}

func (r *httpRouter) Match(req *http.Request) (string, *HTTPRoute) {
	for id, route := range r.routes {
		if route.Rule.Match(req) {
			return id, route
		}
	}

	return "", nil
}

func (r *httpRouter) SetMiddleware(middleware Middleware) HTTPRouter {
	r.middleware = middleware
	return r
}

func (r *httpRouter) Middleware() Middleware {
	return r.middleware
}

func (r *httpRouter) RegisterRoute(routeId string, route *HTTPRoute) HTTPRouter {
	r.routes[routeId] = route
	return r
}

func (r *httpRouter) DeregisterRoute(routeId string) {
	delete(r.routes, routeId)
}

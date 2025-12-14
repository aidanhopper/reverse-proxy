package proxy

import "net/http"

type HTTPRouter interface {
	Match(req *http.Request) *HTTPRoute
	RegisterRoute(routeId string, route *HTTPRoute) HTTPRouter
	DeregisterRoute(routeId string, route *HTTPRoute)
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

func (r *httpRouter) Match(req *http.Request) *HTTPRoute {
	for _, route := range r.routes {
		if route.Rule(req) {
			return route
		}
	}

	return nil
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

func (r *httpRouter) DeregisterRoute(routeId string, route *HTTPRoute) {
	delete(r.routes, routeId)
}

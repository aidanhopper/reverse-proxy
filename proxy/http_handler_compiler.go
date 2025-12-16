package proxy

import (
	"log"
	"net/http"
)

type HTTPHandlerCompiler struct {
	routers  map[string]HTTPRouter
	services map[string]http.Handler
}

func NewHTTPHandlerCompiler() *HTTPHandlerCompiler {
	return &HTTPHandlerCompiler{
		routers:  make(map[string]HTTPRouter),
		services: make(map[string]http.Handler),
	}
}

func (c *HTTPHandlerCompiler) RegisterService(serviceId string, service http.Handler) *HTTPHandlerCompiler {
	c.services[serviceId] = service
	return c
}

func (c *HTTPHandlerCompiler) DeregisterService(serviceId string) {
	delete(c.services, serviceId)
}

func (c *HTTPHandlerCompiler) Service(serviceId string) http.Handler {
	return c.services[serviceId]
}

func (c *HTTPHandlerCompiler) Router(routerId string) HTTPRouter {
	return c.routers[routerId]
}

func (c *HTTPHandlerCompiler) RegisterRouter(routerId string) HTTPRouter {
	router := NewHTTPRouter()
	c.routers[routerId] = router
	return router
}

func (c *HTTPHandlerCompiler) DeregisterRouter(routerId string, router HTTPRouter) {
	delete(c.routers, routerId)
}

func (c *HTTPHandlerCompiler) Compile(routerIds ...string) http.Handler {
	type routerWrapper struct {
		Router HTTPRouter
		Id     string
	}

	var routers []routerWrapper

	for _, id := range routerIds {
		router, ok := c.routers[id]
		if ok {
			routers = append(routers, routerWrapper{
				Router: router,
				Id:     id,
			})
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var route *HTTPRoute
		var router HTTPRouter
		var routerId string
		var routeId string

		for _, rw := range routers {
			routeId, route = rw.Router.Match(r)
			router = rw.Router
			routerId = rw.Id
			if route != nil {
				break
			}
		}

		if route == nil {
			http.NotFound(w, r)
			return
		}

		log.Printf(
			"%s | HTTP router \"%s\" routing request to \"%s\"\n",
			r.RemoteAddr,
			routerId,
			routeId,
		)

		service, ok := c.services[route.ServiceId]
		if !ok {
			http.Error(w, "service not available", http.StatusBadGateway)
			return
		}

		log.Printf(
			"%s | \"%s\" serving \"%s\" service",
			r.RemoteAddr,
			routeId,
			route.ServiceId,
		)

		if router.Middleware() != nil && route.Middleware != nil {
			router.Middleware().Wrap(route.Middleware.Wrap(service)).ServeHTTP(w, r)
			return
		} else if router.Middleware() != nil {
			router.Middleware().Wrap(service).ServeHTTP(w, r)
		} else if route.Middleware != nil {
			route.Middleware.Wrap(service).ServeHTTP(w, r)
		} else {
			service.ServeHTTP(w, r)
		}
	})
}

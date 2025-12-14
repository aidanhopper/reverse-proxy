package proxy

import "net/http"

type HTTPHandleCompiler struct {
	routers  map[string]HTTPRouter
	services map[string]http.Handler
}

func NewHTTPHandleCompiler() *HTTPHandleCompiler {
	return &HTTPHandleCompiler{
		routers:  make(map[string]HTTPRouter),
		services: make(map[string]http.Handler),
	}
}

func (c *HTTPHandleCompiler) RegisterService(serviceId string, service http.Handler) *HTTPHandleCompiler {
	c.services[serviceId] = service
	return c
}

func (c *HTTPHandleCompiler) DeregisterService(serviceId string) {
	delete(c.services, serviceId)
}

func (c *HTTPHandleCompiler) Service(serviceId string) http.Handler {
	return c.services[serviceId]
}

func (c *HTTPHandleCompiler) Router(routerId string) HTTPRouter {
	return c.routers[routerId]
}

func (c *HTTPHandleCompiler) RegisterRouter(routerId string) HTTPRouter {
	router := NewHTTPRouter()
	c.routers[routerId] = router
	return router
}

func (c *HTTPHandleCompiler) DeregisterRouter(routerId string, router HTTPRouter) {
	delete(c.routers, routerId)
}

func (c *HTTPHandleCompiler) Compile(routerId string) http.Handler {
	router, ok := c.routers[routerId]
	if !ok {
		return http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := router.Match(r)
		if route == nil {
			http.NotFound(w, r)
			return
		}

		service, ok := c.services[route.ServiceId]
		if !ok {
			http.Error(w, "service not available", http.StatusBadGateway)
			return
		}

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

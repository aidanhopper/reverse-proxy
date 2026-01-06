package engine

import "log"

type TCPHandler interface {
	ServeTCP(*BufferedTCPConn)
	Rule() Rule
}

type TCPServiceFunc func(conn *BufferedTCPConn)

type tcpHandlerWrapper struct {
	f func(conn *BufferedTCPConn)
	r Rule
}

func TCPHandlerFunc(f func(conn *BufferedTCPConn), rule Rule) TCPHandler {
	return &tcpHandlerWrapper{
		f: f,
		r: rule,
	}
}

func (w *tcpHandlerWrapper) ServeTCP(conn *BufferedTCPConn) {
	w.f(conn)
}

func (w *tcpHandlerWrapper) Rule() Rule {
	return w.r
}

type TCPRoute struct {
	Rule      Rule
	ServiceId string
}

type TCPRouter interface {
	Match(*TCPContext) (string, *TCPRoute)
	RegisterRoute(routeId string, route *TCPRoute) TCPRouter
	DeregisterRoute(routeId string)
	Routes() []*TCPRoute
}

type tcpRouter struct {
	routes map[string]*TCPRoute
}

func NewTCPRouter() TCPRouter {
	return &tcpRouter{
		make(map[string]*TCPRoute),
	}
}

type TCPHandlerCompiler struct {
	routers  map[string]TCPRouter
	services map[string]TCPServiceFunc
}

func NewTCPHandlerCompiler() *TCPHandlerCompiler {
	return &TCPHandlerCompiler{
		routers:  make(map[string]TCPRouter),
		services: make(map[string]TCPServiceFunc),
	}
}

func (c *TCPHandlerCompiler) RegisterService(serviceId string, service func(conn *BufferedTCPConn)) *TCPHandlerCompiler {
	c.services[serviceId] = service
	return c
}

func (c *TCPHandlerCompiler) DeregisterService(serviceId string) {
	delete(c.services, serviceId)
}

func (c *TCPHandlerCompiler) RegisterRouter(routerId string) TCPRouter {
	router := NewTCPRouter()
	c.routers[routerId] = router
	return router
}

func (c *TCPHandlerCompiler) DeregisterRouter(routerId string) {
	delete(c.routers, routerId)
}

func (c *TCPHandlerCompiler) Router(routerId string) TCPRouter {
	if router, ok := c.routers[routerId]; ok {
		return router
	}
	return nil
}

func (r *tcpRouter) Match(ctx *TCPContext) (string, *TCPRoute) {
	for id, route := range r.routes {
		if route.Rule.Match(ctx) {
			return id, route
		}
	}
	return "", nil
}

func (r *tcpRouter) RegisterRoute(routeId string, route *TCPRoute) TCPRouter {
	r.routes[routeId] = route
	return r
}

func (r *tcpRouter) DeregisterRoute(routeId string) {
	delete(r.routes, routeId)
}

func (c *TCPHandlerCompiler) Compile(routerIds ...string) TCPHandler {
	type routerWrapper struct {
		Router TCPRouter
		Id     string
	}

	var routers []routerWrapper
	var rules []Rule

	for _, id := range routerIds {
		router, ok := c.routers[id]
		if !ok {
			continue
		}

		routers = append(routers, routerWrapper{
			Router: router,
			Id:     id,
		})

		for _, route := range router.Routes() {
			rules = append(rules, route.Rule)
		}
	}

	return TCPHandlerFunc(func(conn *BufferedTCPConn) {
		var route *TCPRoute
		var routerId string
		var routeId string

		ctx := NewTCPContext(conn)

		for _, rw := range routers {
			id, r := rw.Router.Match(ctx)
			if r != nil {
				route = r
				routeId = id
				routerId = rw.Id
				break
			}
		}

		if route == nil {
			return
		}

		log.Printf(
			"%s | TCP router \"%s\" routing request to \"%s\"\n",
			conn.RemoteAddr(),
			routerId,
			routeId,
		)

		service, ok := c.services[route.ServiceId]

		if !ok {
			return
		}

		log.Printf(
			"%s | \"%s\" serving \"%s\" service",
			conn.RemoteAddr(),
			routeId,
			route.ServiceId,
		)

		service(conn)

		conn.Close()
	}, Or(rules...))
}

func (r *tcpRouter) Routes() []*TCPRoute {
	var routes []*TCPRoute
	for _, route := range r.routes {
		routes = append(routes, route)
	}
	return routes
}

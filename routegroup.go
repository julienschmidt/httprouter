package httprouter

import (
	"net/http"
)

type RouteGroup struct {
	r *Router
	p string
}

func newRouteGroup(r *Router, path string) *RouteGroup {
	if path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	//Strip traling / (if present) as all added sub paths must start with a /
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return &RouteGroup{r: r, p: path}
}

func (r *RouteGroup) NewGroup(path string) *RouteGroup {
	return newRouteGroup(r.r, r.subPath(path))
}

func (r *RouteGroup) Handle(method, path string, handle Handle) {
	r.r.Handle(method, r.subPath(path), handle)
}

func (r *RouteGroup) Handler(method, path string, handler http.Handler) {
	r.r.Handler(method, r.subPath(path), handler)
}

func (r *RouteGroup) HandlerFunc(method, path string, handler http.HandlerFunc) {
	r.r.HandlerFunc(method, r.subPath(path), handler)
}

func (r *RouteGroup) GET(path string, handle Handle) {
	r.Handle("GET", path, handle)
}
func (r *RouteGroup) HEAD(path string, handle Handle) {
	r.Handle("HEAD", path, handle)
}
func (r *RouteGroup) OPTIONS(path string, handle Handle) {
	r.Handle("OPTIONS", path, handle)
}
func (r *RouteGroup) POST(path string, handle Handle) {
	r.Handle("POST", path, handle)
}
func (r *RouteGroup) PUT(path string, handle Handle) {
	r.Handle("PUT", path, handle)
}
func (r *RouteGroup) PATCH(path string, handle Handle) {
	r.Handle("PATCH", path, handle)
}
func (r *RouteGroup) DELETE(path string, handle Handle) {
	r.Handle("DELETE", path, handle)
}

func (r *RouteGroup) subPath(path string) string {
	if path[0] != '/' {
		panic("path must start with a '/'")
	}
	return r.p + path
}

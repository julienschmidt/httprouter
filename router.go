// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// A radix tree based HTTP request router
package httprouter

import (
	"errors"
	"net/http"
)

// Handle is a function that can be registered to a route to handle HTTP
// requests. Like http.HandlerFunc, but has a third parameter for the route
// parameters.
type Handle func(http.ResponseWriter, *http.Request, map[string]string)

// NotFound is the default HTTP handler func for routes that can't be matched
// with an existing route.
// NotFound tries to redirect to a canonical URL generated with CleanPath
func NotFound(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		if p := CleanPath(req.URL.Path); p != req.URL.Path && p != req.Referer() {
			http.Redirect(rw, req, p, http.StatusMovedPermanently)
			return
		}
	}

	http.NotFound(rw, req)
}

// Router is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes
type Router struct {
	node

	// Enables automatic redirection if the current route can't be matched but
	// handler for the path with (without) the trailing slash exists.
	// For example if a route for /foo exists but /foo/ is requested, the client
	// would be redirected to /foo with http status code 301.
	RedirectTrailingSlash bool

	// Configurable handler func which is used when no matching route is found.
	// Default is the NotFound func of this package
	NotFound http.HandlerFunc

	// Handler func to handle panics recovered from http handlers.
	// It should be used to generate a error page and return the http error code
	// "500 - Internal Server Error".
	// The handler can be used to keep your server from crashing because of
	// unrecovered panics.
	PanicHandler func(http.ResponseWriter, *http.Request, interface{})
}

// Make sure the Router conforms with the http.Handler interface
var _ http.Handler = New()

// New returnes a new initialized Router.
// The router can be configured to also match the requested HTTP method or the
// requested Host.
func New() *Router {
	return &Router{
		RedirectTrailingSlash: true,
		NotFound:              NotFound,
	}
}

// GET is a shortcut for router.Handle("GET", path, handle)
func (r *Router) GET(path string, handle Handle) error {
	return r.Handle("GET", path, handle)
}

// POST is a shortcut for router.Handle("POST", path, handle)
func (r *Router) POST(path string, handle Handle) error {
	return r.Handle("POST", path, handle)
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func (r *Router) PUT(path string, handle Handle) error {
	return r.Handle("PUT", path, handle)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (r *Router) DELETE(path string, handle Handle) error {
	return r.Handle("DELETE", path, handle)
}

// Handle registers a new request handle with the given path and method.
func (r *Router) Handle(method, path string, handle Handle) error {
	if path[0] != '/' {
		return errors.New("Path must begin with /")
	}
	return r.addRoute(method, path, handle)
}

// Make the router implement the http.Handler interface.
func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer func() {
			if rcv := recover(); rcv != nil {
				r.PanicHandler(rw, req, rcv)
			}
		}()
	}

	path := req.URL.Path

	if handle, vars, tsr := r.getValue(req.Method, path); handle != nil {
		handle(rw, req, vars)
	} else if tsr && r.RedirectTrailingSlash {
		if path[len(path)-1] == '/' {
			http.Redirect(rw, req, path[:len(path)-1], http.StatusMovedPermanently)
			return
		} else {
			http.Redirect(rw, req, path+"/", http.StatusMovedPermanently)
			return
		}
	} else { // Handle 404
		r.NotFound(rw, req)
	}
}

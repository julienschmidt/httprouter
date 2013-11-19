// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// A radix tree based HTTP request router
package router

import (
	"errors"
	"net/http"
)

// HandlerFunc is a function that can be registered to a route to handle HTTP
// requests. Like http.HandlerFunc, but has a third parameter for the route
// parameters.
type HandlerFunc func(http.ResponseWriter, *http.Request, map[string]string)

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
		NotFound:              http.NotFound,
	}
}

// Add registers a new request handler to the given path.
func (r *Router) Add(path string, h HandlerFunc) error {
	if path[0] != '/' {
		return errors.New("Path must begin with /")
	}
	return r.addRoute(path, h)
}

// Make the router implement the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer func() {
			if rcv := recover(); rcv != nil {
				r.PanicHandler(w, req, rcv)
			}
		}()
	}

	path := req.URL.Path

	if handle, vars, tsr := r.getValue(path); handle != nil {
		handle(w, req, vars)
	} else if tsr && r.RedirectTrailingSlash {
		if path[len(path)-1] == '/' {
			http.Redirect(w, req, path[:len(path)-1], http.StatusMovedPermanently)
			return
		} else {
			http.Redirect(w, req, path+"/", http.StatusMovedPermanently)
			return
		}
	} else { // Handle 404
		r.NotFound(w, req)
	}
}

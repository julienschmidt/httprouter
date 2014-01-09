# HttpRouter [![Build Status](https://travis-ci.org/julienschmidt/httprouter.png?branch=master)](https://travis-ci.org/julienschmidt/httprouter) [![Codebot](https://codebot.io/badge/github.com/julienschmidt/httprouter.png)](http://codebot.io/doc/pkg/github.com/julienschmidt/httprouter "Codebot") [![GoDoc](http://godoc.org/github.com/julienschmidt/httprouter?status.png)](http://godoc.org/github.com/julienschmidt/httprouter)

HttpRouter is a high performance HTTP request router
(also called *multiplexer* or just *mux* for short).

In contrast to the default mux of Go's net/http package, this router supports
variables in the routing pattern and matches against the request method.
It also scales better.

The router is optimized for best performance and a small memory footprint.
It scales well even with very long pathes and a large number of routes.
A compressing dynamic trie (radix tree) structure is used for efficient matching.

## Features
**Zero Garbage:** The matching and dispatching process generates zero bytes of garbage. In fact, the only heap allocations that are made, is by building the map containing the variables key-value pairs. If the request path contains no variables, not a single heap allocation is necessary.

**Variables in your routing pattern:** Stop parsing the requested URL path, just give the path segment a name and the router delivers the value to you. Because of the design of the router, pattern variables are very cheap.

**Only explicit matches:** With other routers / muxes, like [http.ServeMux](http://golang.org/pkg/net/http/#ServeMux), a requested URL path could match multiple patterns. Therefore they have some awkward pattern priority rules, like *longest match* or *first registered, first matched*. By design of this router, a request can only match exactly one or no route. As a result, there are also no unintended matches, which makes it great for SEO. 

**Stop caring about trailing slashes:** Choose the URL style you like, the router automatically redirects the client if a trailing slash is missing or if there is one extra. Of course it only does so, if the new path has a handler. If you don't like it, you can turn off this behavior.

**No more server crashes:** You can set a PanicHandler to deal with panics. The router then recovers and lets the PanicHandler log what happened and deliver a nice error page.

Of course you can also set a custom NotFound handler and serve files.

## Usage
This is just a quick introduction, view the [GoDoc](http://godoc.org/github.com/julienschmidt/httprouter) for details.

Let's start with a trivial example:
```go
package main

import (
    "fmt"
    "github.com/julienschmidt/httprouter"
    "net/http"
    "log"
)

func Index(w http.ResponseWriter, r *http.Request, _ map[string]string) {
    fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, vars map[string]string) {
    fmt.Fprintf(w, "hello, %s!\n", vars["name"])
}

func main() {
    router := httprouter.New()
    router.GET("/", Index)
    router.GET("/hello/:name", Hello)

    log.Fatal(http.ListenAndServe(":12345", router))
}
```

### Named parameters
As you can see, `:name` is a *named parameter*.
The values are passed in map, therefore the value of `:name` is available in `vars["name"]`.

Named parameters only match a single path segment:
```
Pattern: /user/:user

 /user/gordon              match
 /user/you                 match
 /user/gordon/profile      no match
 /user/                    no match
```

**Note:** Since this router has only explicit matches, you can not register static routes and paramters for the same path segment. For example you can not register the patterns `/user/new` and `/user/:user` at the same time.

### Catch-All routes
The second type are *catch-all* routes and have the form `*name`. Like the name suggest, they match everything.
Therefore they must always be at the **end** of the pattern:
```
Pattern: /src/*filepath

 /src/                     match
 /src/somefile.go          match
 /src/subdir/somefile.go   match
```

## Where can I find Middleware *X*?
This package just provides a very efficient request router with a few extra features. The router is just a [http.Handler](http://golang.org/pkg/net/http/#Handler), you can chain any http.Handler compatible middleware before the router, for example the [Gorilla handlers](http://www.gorillatoolkit.org/pkg/handlers). Or you could [just write your own](http://justinas.org/writing-http-middleware-in-go/), it's very easy!

Here is a quick example: Does your server serve multiple domains / hosts? You want to use subdomains?
Define a router per host!
```go
type HostSwitch map[string]http.Handler

func (hs HostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler := hs[r.URL.Host]; handler != nil {
		handler.ServeHTTP(w, r)
	}

	http.Error(w, "Forbidden", 403) // Or Redirect?
}

func main() {
    router := httprouter.New()
    router.GET("/", Index)
    router.GET("/hello/:name", Hello)

    hs := make(HostSwitch)
	hs["example.com"] = router

    log.Fatal(http.ListenAndServe(":12345", hs))
}
```

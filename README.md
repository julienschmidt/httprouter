# FastHttpRouter
[![Build Status](https://travis-ci.org/buaazp/fasthttprouter.svg?branch=master)](https://travis-ci.org/buaazp/fasthttprouter)
[![Coverage Status](https://coveralls.io/repos/buaazp/fasthttprouter/badge.svg?branch=master&service=github)](https://coveralls.io/github/buaazp/fasthttprouter?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/buaazp/fasthttprouter)](https://goreportcard.com/report/github.com/buaazp/fasthttprouter)
[![GoDoc](http://godoc.org/github.com/buaazp/fasthttprouter?status.svg)](http://godoc.org/github.com/buaazp/fasthttprouter)
[![GitHub release](https://img.shields.io/github/release/buaazp/fasthttprouter.svg)](https://github.com/buaazp/fasthttprouter/releases)

FastHttpRouter is forked from [httprouter](https://github.com/julienschmidt/httprouter) which is a lightweight high performance HTTP request router
(also called *multiplexer* or just *mux* for short) for [fasthttp](https://github.com/valyala/fasthttp).

This router is optimized for high performance and a small memory footprint. It scales well even with very long paths and a large number of routes. A compressing dynamic trie (radix tree) structure is used for efficient matching.

#### License Related

- The author of `httprouter` [@julienschmidt](https://github.com/julienschmidt) did almost all the hard work of this router.
- I respect the laws of open source. So LICENSE of `httprouter` is alway stay here: [HttpRouterLicense](HttpRouterLicense).
- What I do is just fit for `fasthttp`. I have no hope to build a huge but toxic go web framwork like [iris](https://github.com/kataras/iris). 
- I fork this repo is just because there is no router for `fasthttp` at that time. And `fasthttprouter` is the FIRST router for `fasthttp`. 
- `fasthttprouter` has been used in my online production and processes 17 million requests per day. It is fast and stable, so I decide to release a stable version.

#### Releases

- [2016.10.24] [v0.1.0](https://github.com/buaazp/fasthttprouter/releases/tag/v0.1.0) The first release version of `fasthttprouter`.

## Features

**Best Performance:** FastHttpRouter is **one of the fastest** go web frameworks in the [go-web-framework-benchmark](https://github.com/smallnest/go-web-framework-benchmark). Even faster than httprouter itself.

- Basic Test: The first test case is to mock 0 ms, 10 ms, 100 ms, 500 ms processing time in handlers. The concurrency clients are 5000.

![](http://ww3.sinaimg.cn/large/4c422e03jw1f2p6nyqh9ij20mm0aktbj.jpg)

- Concurrency Test: In 30 ms processing time, the tets result for 100, 1000, 5000 clients is:

![](http://ww4.sinaimg.cn/large/4c422e03jw1f2p6o1cdbij20lk09sack.jpg)

See below for technical details of the implementation.

**Only explicit matches:** With other routers, like [http.ServeMux](http://golang.org/pkg/net/http/#ServeMux),
a requested URL path could match multiple patterns. Therefore they have some
awkward pattern priority rules, like *longest match* or *first registered,
first matched*. By design of this router, a request can only match exactly one
or no route. As a result, there are also no unintended matches, which makes it
great for SEO and improves the user experience.

**Stop caring about trailing slashes:** Choose the URL style you like, the
router automatically redirects the client if a trailing slash is missing or if
there is one extra. Of course it only does so, if the new path has a handler.
If you don't like it, you can [turn off this behavior](http://godoc.org/github.com/buaazp/fasthttprouter#Router.RedirectTrailingSlash).

**Path auto-correction:** Besides detecting the missing or additional trailing
slash at no extra cost, the router can also fix wrong cases and remove
superfluous path elements (like `../` or `//`).
Is [CAPTAIN CAPS LOCK](http://www.urbandictionary.com/define.php?term=Captain+Caps+Lock) one of your users?
FastHttpRouter can help him by making a case-insensitive look-up and redirecting him
to the correct URL.

**Parameters in your routing pattern:** Stop parsing the requested URL path,
just give the path segment a name and the router delivers the dynamic value to
you. Because of the design of the router, path parameters are very cheap.

**Zero Garbage:** The matching and dispatching process generates zero bytes of
garbage. In fact, the only heap allocations that are made, is by building the
slice of the key-value pairs for path parameters. If the request path contains
no parameters, not a single heap allocation is necessary.

**No more server crashes:** You can set a [Panic handler](http://godoc.org/github.com/buaazp/fasthttprouter#Router.PanicHandler) to deal with panics
occurring during handling a HTTP request. The router then recovers and lets the
PanicHandler log what happened and deliver a nice error page.

**Perfect for APIs:** The router design encourages to build sensible, hierarchical
RESTful APIs. Moreover it has builtin native support for [OPTIONS requests](http://zacstewart.com/2012/04/14/http-options-method.html)
and `405 Method Not Allowed` replies.

Of course you can also set **custom [NotFound](http://godoc.org/github.com/buaazp/fasthttprouter#Router.NotFound) and  [MethodNotAllowed](http://godoc.org/github.com/buaazp/fasthttprouter#Router.MethodNotAllowed) handlers** and [**serve static files**](http://godoc.org/github.com/buaazp/fasthttprouter#Router.ServeFiles).

## Usage

This is just a quick introduction, view the [GoDoc](http://godoc.org/github.com/buaazp/fasthttprouter) for details:

Let's start with a trivial example:

```go
package main

import (
	"fmt"
	"log"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

func Index(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "Welcome!\n")
}

func Hello(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "hello, %s!\n", ctx.UserValue("name"))
}

func main() {
	router := fasthttprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)

	log.Fatal(fasthttp.ListenAndServe(":8080", router.Handler))
}
```

### Named parameters

As you can see, `:name` is a *named parameter*. The values are accessible via `RequestCtx.UserValues`. You can get the value of a parameter by using the `ctx.UserValue("name")`.

Named parameters only match a single path segment:

```
Pattern: /user/:user

 /user/gordon              match
 /user/you                 match
 /user/gordon/profile      no match
 /user/                    no match
```

**Note:** Since this router has only explicit matches, you can not register static routes and parameters for the same path segment. For example you can not register the patterns `/user/new` and `/user/:user` for the same request method at the same time. The routing of different request methods is independent from each other.

### Catch-All parameters

The second type are *catch-all* parameters and have the form `*name`.
Like the name suggests, they match everything.
Therefore they must always be at the **end** of the pattern:

```
Pattern: /src/*filepath

 /src/                     match
 /src/somefile.go          match
 /src/subdir/somefile.go   match
```

## How does it work?

The router relies on a tree structure which makes heavy use of *common prefixes*, it is basically a *compact* [*prefix tree*](https://en.wikipedia.org/wiki/Trie) (or just [*Radix tree*](https://en.wikipedia.org/wiki/Radix_tree)). Nodes with a common prefix also share a common parent. Here is a short example what the routing tree for the `GET` request method could look like:

```
Priority   Path             Handle
9          \                *<1>
3          ├s               nil
2          |├earch\         *<2>
1          |└upport\        *<3>
2          ├blog\           *<4>
1          |    └:post      nil
1          |         └\     *<5>
2          ├about-us\       *<6>
1          |        └team\  *<7>
1          └contact\        *<8>
```

Every `*<num>` represents the memory address of a handler function (a pointer). If you follow a path trough the tree from the root to the leaf, you get the complete route path, e.g `\blog\:post\`, where `:post` is just a placeholder ([*parameter*](#named-parameters)) for an actual post name. Unlike hash-maps, a tree structure also allows us to use dynamic parts like the `:post` parameter, since we actually match against the routing patterns instead of just comparing hashes. [As benchmarks show][benchmark], this works very well and efficient.

Since URL paths have a hierarchical structure and make use only of a limited set of characters (byte values), it is very likely that there are a lot of common prefixes. This allows us to easily reduce the routing into ever smaller problems. Moreover the router manages a separate tree for every request method. For one thing it is more space efficient than holding a method->handle map in every single node, for another thing is also allows us to greatly reduce the routing problem before even starting the look-up in the prefix-tree.

For even better scalability, the child nodes on each tree level are ordered by priority, where the priority is just the number of handles registered in sub nodes (children, grandchildren, and so on..). This helps in two ways:

1. Nodes which are part of the most routing paths are evaluated first. This helps to make as much routes as possible to be reachable as fast as possible.
2. It is some sort of cost compensation. The longest reachable path (highest cost) can always be evaluated first. The following scheme visualizes the tree structure. Nodes are evaluated from top to bottom and from left to right.

```
├------------
├---------
├-----
├----
├--
├--
└-
```

## Why doesn't this work with `http.Handler`?

Becasue fasthttp doesn't provide http.Handler. See this [description](https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp).

Fasthttp works with [RequestHandler](https://godoc.org/github.com/valyala/fasthttp#RequestHandler) functions instead of objects implementing Handler interface. So a FastHttpRouter provides a [Handler](https://godoc.org/github.com/buaazp/fasthttprouter#Router.Handler) interface to implement the fasthttp.ListenAndServe interface.

Just try it out for yourself, the usage of FastHttpRouter is very straightforward. The package is compact and minimalistic, but also probably one of the easiest routers to set up.

## Where can I find Middleware *X*?

This package just provides a very efficient request router with a few extra features. The router is just a [`fasthttp.RequestHandler`](https://godoc.org/github.com/valyala/fasthttp#RequestHandler), you can chain any `fasthttp.RequestHandler` compatible middleware before the router. Or you could [just write your own](https://justinas.org/writing-http-middleware-in-go/), it's very easy!

Have a look at these midware examples:

- [Auth Midware](examples/auth)
- [Multi Hosts Midware](examples/hosts)

## Chaining with the NotFound handler

**NOTE: It might be required to set [Router.HandleMethodNotAllowed](http://godoc.org/github.com/buaazp/fasthttprouter#Router.HandleMethodNotAllowed) to `false` to avoid problems.**

You can use another [http.Handler](http://golang.org/pkg/net/http/#Handler), for example another router, to handle requests which could not be matched by this router by using the [Router.NotFound](http://godoc.org/github.com/buaazp/fasthttprouter#Router.NotFound) handler. This allows chaining.

### Static files
The `NotFound` handler can for example be used to serve static files from the root path `/` (like an index.html file along with other assets):

```go
// Serve static files from the ./public directory
router.NotFound = fasthttp.FSHandler("./public", 0)
```

But this approach sidesteps the strict core rules of this router to avoid routing problems. A cleaner approach is to use a distinct sub-path for serving files, like `/static/*filepath` or `/files/*filepath`.

## Web Frameworks based on FastHttpRouter

If the HttpRouter is a bit too minimalistic for you, you might try one of the following more high-level 3rd-party web frameworks building upon the HttpRouter package:

- Waiting for you to do this...

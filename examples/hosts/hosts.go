package main

import (
	"fmt"
	"log"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "Welcome!\n")
}

// Hello is the Hello handler
func Hello(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "hello, %s!\n", ctx.UserValue("name"))
}

// HostSwitch is the host-handler map
// We need an object that implements the fasthttp.RequestHandler interface.
// We just use a map here, in which we map host names (with port) to fasthttp.RequestHandlers
type HostSwitch map[string]fasthttp.RequestHandler

// CheckHost Implement a CheckHost method on our new type
func (hs HostSwitch) CheckHost(ctx *fasthttp.RequestCtx) {
	// Check if a http.Handler is registered for the given host.
	// If yes, use it to handle the request.
	if handler := hs[string(ctx.Host())]; handler != nil {
		handler(ctx)
	} else {
		// Handle host names for wich no handler is registered
		ctx.Error("Forbidden", 403) // Or Redirect?
	}
}

func main() {
	// Initialize a router as usual
	router := fasthttprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)

	// Make a new HostSwitch and insert the router (our http handler)
	// for example.com and port 12345
	hs := make(HostSwitch)
	hs["example.com:12345"] = router.Handler

	// Use the HostSwitch to listen and serve on port 12345
	log.Fatal(fasthttp.ListenAndServe(":12345", hs.CheckHost))
}

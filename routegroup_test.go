package httprouter

import (
	"net/http"
	"testing"
)

func TestRouteGroupOfARouteGroup(t *testing.T) {
	var get bool
	router := New()
	foo := router.NewGroup("/foo") // creates /foo group
	bar := foo.NewGroup("/bar")

	bar.GET("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		get = true
	})

	w := new(mockResponseWriter)

	r, _ := http.NewRequest("GET", "/foo/bar/GET", nil)
	router.ServeHTTP(w, r)
	if !get {
		t.Error("routing GET /foo/bar/GET failed")
	}

}

func TestRouteGroupAPI(t *testing.T) {
	var get, head, options, post, put, patch, delete, handler, handlerFunc bool

	httpHandler := handlerStruct{&handler}

	router := New()
	group := router.NewGroup("/foo") // creates /foo group

	group.GET("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		get = true
	})
	group.HEAD("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		head = true
	})
	group.OPTIONS("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		options = true
	})
	group.POST("/POST", func(w http.ResponseWriter, r *http.Request, _ Params) {
		post = true
	})
	group.PUT("/PUT", func(w http.ResponseWriter, r *http.Request, _ Params) {
		put = true
	})
	group.PATCH("/PATCH", func(w http.ResponseWriter, r *http.Request, _ Params) {
		patch = true
	})
	group.DELETE("/DELETE", func(w http.ResponseWriter, r *http.Request, _ Params) {
		delete = true
	})
	group.Handler("GET", "/Handler", httpHandler)
	group.HandlerFunc("GET", "/HandlerFunc", func(w http.ResponseWriter, r *http.Request) {
		handlerFunc = true
	})

	w := new(mockResponseWriter)

	r, _ := http.NewRequest("GET", "/foo/GET", nil)
	router.ServeHTTP(w, r)
	if !get {
		t.Error("routing /foo/GET failed")
	}

	r, _ = http.NewRequest("HEAD", "/foo/GET", nil)
	router.ServeHTTP(w, r)
	if !head {
		t.Error("routing HEAD failed")
	}

	r, _ = http.NewRequest("OPTIONS", "/foo/GET", nil)
	router.ServeHTTP(w, r)
	if !options {
		t.Error("routing OPTIONS failed")
	}

	r, _ = http.NewRequest("POST", "/foo/POST", nil)
	router.ServeHTTP(w, r)
	if !post {
		t.Error("routing POST failed")
	}

	r, _ = http.NewRequest("PUT", "/foo/PUT", nil)
	router.ServeHTTP(w, r)
	if !put {
		t.Error("routing PUT failed")
	}

	r, _ = http.NewRequest("PATCH", "/foo/PATCH", nil)
	router.ServeHTTP(w, r)
	if !patch {
		t.Error("routing PATCH failed")
	}

	r, _ = http.NewRequest("DELETE", "/foo/DELETE", nil)
	router.ServeHTTP(w, r)
	if !delete {
		t.Error("routing DELETE failed")
	}

	r, _ = http.NewRequest("GET", "/foo/Handler", nil)
	router.ServeHTTP(w, r)
	if !handler {
		t.Error("routing Handler failed")
	}

	r, _ = http.NewRequest("GET", "/foo/HandlerFunc", nil)
	router.ServeHTTP(w, r)
	if !handlerFunc {
		t.Error("routing HandlerFunc failed")
	}
}

// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"

	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func newRequest(t *testing.T, method, requestURI string) *http.Request {
	buf := bytes.NewBuffer(nil)
	reader := bufio.NewReader(buf)
	buf.WriteString(fmt.Sprintf("%s %s HTTP/1.0\n", method, requestURI))
	buf.WriteString("\n")
	req, err := http.ReadRequest(reader)
	if err != nil {
		req, err = http.NewRequest(method, requestURI, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	return req
}

func TestParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	for i := range ps {
		if val := ps.ByName(ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := ps.ByName("noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func TestRouter(t *testing.T) {
	router := New()

	routed := false
	router.Handle("GET", "/user/:name", func(w http.ResponseWriter, r *http.Request, ps Params) {
		routed = true
		want := Params{Param{"name", "gopher"}}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
	})

	w := new(mockResponseWriter)

	req := newRequest(t, "GET", "/user/gopher")
	router.ServeHTTP(w, req)

	if !routed {
		t.Fatal("routing failed")
	}
}

func TestRawPathRouting(t *testing.T) {
	router := New()
	router.RawPathRouting = true

	routed := false
	router.Handle("GET", "/path/:id", func(w http.ResponseWriter, r *http.Request, ps Params) {
		routed = true
		want := Params{Param{"id", "go%2fpher"}}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
	})

	w := new(mockResponseWriter)

	req := newRequest(t, "GET", "/path/go%2fpher")
	router.ServeHTTP(w, req)
	if !routed {
		t.Fatal("routing failed")
	}
}

func TestRawPathRoutingMixed(t *testing.T) {
	router := New()
	router.RawPathRouting = true

	routed := false
	router.Handle("GET", "/u/:u/pher/p/:p", func(w http.ResponseWriter, r *http.Request, ps Params) {
		routed = true
		want := Params{Param{"u", "go%2fpher"}, Param{"p", "pher%2fgo"}}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
	})

	w := new(mockResponseWriter)

	req := newRequest(t, "GET", "/u/go%2fpher/pher/p/pher%2fgo")
	router.ServeHTTP(w, req)
	if !routed {
		t.Fatal("routing failed")
	}
}

func TestRawPathRoutingCleanPath(t *testing.T) {
	router := New()
	router.RawPathRouting = true

	routed := false
	router.Handle("GET", "/u/:u/pher/p/:p", func(w http.ResponseWriter, r *http.Request, ps Params) {
		routed = true
		want := Params{Param{"u", "."}, Param{"p", ".."}}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
	})

	w := new(mockResponseWriter)

	req := newRequest(t, "GET", "/u/./pher/p/..")
	router.ServeHTTP(w, req)
	if !routed {
		t.Fatal("routing failed")
	}
}

func TestRawPathRoutingNotFound(t *testing.T) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	router := New()
	router.RawPathRouting = true
	router.GET("/path/:id", handlerFunc)
	router.GET("/dir/:id/", handlerFunc)

	testRoutes := []struct {
		route  string
		code   int
		header string
	}{
		{"/path/go%2fpher/", 301, "map[Location:[/path/go%2fpher]]"},   // TSR -/
		{"/dir/go%2fpher", 301, "map[Location:[/dir/go%2fpher/]]"},     // TSR +/
		{"/PATH/go%2fpher", 301, "map[Location:[/path/go%2fpher]]"},    // Fixed Case
		{"/DIR/go%2fpher/", 301, "map[Location:[/dir/go%2fpher/]]"},    // Fixed Case
		{"/PATH/go%2fpher/", 301, "map[Location:[/path/go%2fpher]]"},   // Fixed Case -/
		{"/DIR/go%2fpher", 301, "map[Location:[/dir/go%2fpher/]]"},     // Fixed Case +/
		{"/../path/go%2fpher", 301, "map[Location:[/path/go%2fpher]]"}, // CleanPath
		{"/nope", 404, ""},                                             // NotFound
	}
	for _, tr := range testRoutes {
		r := newRequest(t, "GET", tr.route)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if !(w.Code == tr.code && (w.Code == 404 || fmt.Sprint(w.Header()) == tr.header)) {
			t.Errorf("NotFound handling route %s failed: Code=%d, Header=%v", tr.route, w.Code, w.Header())
		}
	}
}

type handlerStruct struct {
	handeled *bool
}

func (h handlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	*h.handeled = true
}

func TestRouterAPI(t *testing.T) {
	var get, head, post, put, patch, delete, handler, handlerFunc bool

	httpHandler := handlerStruct{&handler}

	router := New()
	router.GET("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		get = true
	})
	router.HEAD("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		head = true
	})
	router.POST("/POST", func(w http.ResponseWriter, r *http.Request, _ Params) {
		post = true
	})
	router.PUT("/PUT", func(w http.ResponseWriter, r *http.Request, _ Params) {
		put = true
	})
	router.PATCH("/PATCH", func(w http.ResponseWriter, r *http.Request, _ Params) {
		patch = true
	})
	router.DELETE("/DELETE", func(w http.ResponseWriter, r *http.Request, _ Params) {
		delete = true
	})
	router.Handler("GET", "/Handler", httpHandler)
	router.HandlerFunc("GET", "/HandlerFunc", func(w http.ResponseWriter, r *http.Request) {
		handlerFunc = true
	})

	w := new(mockResponseWriter)

	r := newRequest(t, "GET", "/GET")
	router.ServeHTTP(w, r)
	if !get {
		t.Error("routing GET failed")
	}

	r = newRequest(t, "HEAD", "/GET")
	router.ServeHTTP(w, r)
	if !head {
		t.Error("routing HEAD failed")
	}

	r = newRequest(t, "POST", "/POST")
	router.ServeHTTP(w, r)
	if !post {
		t.Error("routing POST failed")
	}

	r = newRequest(t, "PUT", "/PUT")
	router.ServeHTTP(w, r)
	if !put {
		t.Error("routing PUT failed")
	}

	r = newRequest(t, "PATCH", "/PATCH")
	router.ServeHTTP(w, r)
	if !patch {
		t.Error("routing PATCH failed")
	}

	r = newRequest(t, "DELETE", "/DELETE")
	router.ServeHTTP(w, r)
	if !delete {
		t.Error("routing DELETE failed")
	}

	r = newRequest(t, "GET", "/Handler")
	router.ServeHTTP(w, r)
	if !handler {
		t.Error("routing Handler failed")
	}

	r = newRequest(t, "GET", "/HandlerFunc")
	router.ServeHTTP(w, r)
	if !handlerFunc {
		t.Error("routing HandlerFunc failed")
	}
}

func TestRouterRoot(t *testing.T) {
	router := New()
	recv := catchPanic(func() {
		router.GET("noSlashRoot", nil)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	router := New()
	router.POST("/path", handlerFunc)

	// Test not allowed
	r := newRequest(t, "GET", "/path")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	w = httptest.NewRecorder()
	responseText := "custom method"
	router.MethodNotAllowed = func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(responseText))
	}
	router.ServeHTTP(w, r)
	if got := w.Body.String(); !(got == responseText) {
		t.Errorf("unexpected response got %q want %q", got, responseText)
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("unexpected response code %d want %d", w.Code, http.StatusTeapot)
	}
}

func TestRouterNotFound(t *testing.T) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	router := New()
	router.GET("/path", handlerFunc)
	router.GET("/dir/", handlerFunc)
	router.GET("/", handlerFunc)

	testRoutes := []struct {
		route  string
		code   int
		header string
	}{
		{"/path/", 301, "map[Location:[/path]]"},   // TSR -/
		{"/dir", 301, "map[Location:[/dir/]]"},     // TSR +/
		{"", 301, "map[Location:[/]]"},             // TSR +/
		{"/PATH", 301, "map[Location:[/path]]"},    // Fixed Case
		{"/DIR/", 301, "map[Location:[/dir/]]"},    // Fixed Case
		{"/PATH/", 301, "map[Location:[/path]]"},   // Fixed Case -/
		{"/DIR", 301, "map[Location:[/dir/]]"},     // Fixed Case +/
		{"/../path", 301, "map[Location:[/path]]"}, // CleanPath
		{"/nope", 404, ""},                         // NotFound
	}
	for _, tr := range testRoutes {
		r := newRequest(t, "GET", tr.route)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if !(w.Code == tr.code && (w.Code == 404 || fmt.Sprint(w.Header()) == tr.header)) {
			t.Errorf("NotFound handling route %s failed: Code=%d, Header=%v", tr.route, w.Code, w.Header())
		}
	}

	// Test custom not found handler
	var notFound bool
	router.NotFound = func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(404)
		notFound = true
	}
	r := newRequest(t, "GET", "/nope")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == 404 && notFound == true) {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// Test other method than GET (want 307 instead of 301)
	router.PATCH("/path", handlerFunc)
	r = newRequest(t, "PATCH", "/path/")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == 307 && fmt.Sprint(w.Header()) == "map[Location:[/path]]") {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// Test special case where no node for the prefix "/" exists
	router = New()
	router.GET("/a", handlerFunc)
	r = newRequest(t, "GET", "/")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == 404) {
		t.Errorf("NotFound handling route / failed: Code=%d", w.Code)
	}
}

func TestRouterPanicHandler(t *testing.T) {
	router := New()
	panicHandled := false

	router.PanicHandler = func(rw http.ResponseWriter, r *http.Request, p interface{}) {
		panicHandled = true
	}

	router.Handle("PUT", "/user/:name", func(_ http.ResponseWriter, _ *http.Request, _ Params) {
		panic("oops!")
	})

	w := new(mockResponseWriter)
	req := newRequest(t, "PUT", "/user/gopher")

	defer func() {
		if rcv := recover(); rcv != nil {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, req)

	if !panicHandled {
		t.Fatal("simulating failed")
	}
}

func TestRouterLookup(t *testing.T) {
	routed := false
	wantHandle := func(_ http.ResponseWriter, _ *http.Request, _ Params) {
		routed = true
	}
	wantParams := Params{Param{"name", "gopher"}}

	router := New()

	// try empty router first
	handle, _, tsr := router.Lookup("GET", "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}

	// insert route and try again
	router.GET("/user/:name", wantHandle)

	handle, params, tsr := router.Lookup("GET", "/user/gopher")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(nil, nil, nil)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}

	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, params)
	}

	handle, _, tsr = router.Lookup("GET", "/user/gopher/")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if !tsr {
		t.Error("Got no TSR recommendation!")
	}

	handle, _, tsr = router.Lookup("GET", "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}
}

type mockFileSystem struct {
	opened bool
}

func (mfs *mockFileSystem) Open(name string) (http.File, error) {
	mfs.opened = true
	return nil, errors.New("this is just a mock")
}

func TestRouterServeFiles(t *testing.T) {
	router := New()
	mfs := &mockFileSystem{}

	recv := catchPanic(func() {
		router.ServeFiles("/noFilepath", mfs)
	})
	if recv == nil {
		t.Fatal("registering path not ending with '*filepath' did not panic")
	}

	router.ServeFiles("/*filepath", mfs)
	w := new(mockResponseWriter)
	r := newRequest(t, "GET", "/favicon.ico")
	router.ServeHTTP(w, r)
	if !mfs.opened {
		t.Error("serving file failed")
	}
}

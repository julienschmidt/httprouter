// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func printChildren(n *node, prefix string) {
	fmt.Printf(" %02d %s%s[%d] %v %t \r\n", n.priority, prefix, n.path, len(n.children), n.handle, n.wildChild)
	for l := len(n.path); l > 0; l-- {
		prefix += " "
	}
	for _, child := range n.children {
		printChildren(child, prefix)
	}
}

// Used as a workaround since we can't compare functions or their adresses
var fakeHandlerValue string

func fakeHandler(val string) Handle {
	return func(http.ResponseWriter, *http.Request, map[string]string) {
		fakeHandlerValue = val
	}
}

type testRequests []struct {
	path       string
	nilHandler bool
	route      string
	vars       map[string]string
}

func checkRequests(t *testing.T, tree *node, requests testRequests) {
	for _, request := range requests {
		handler, vars, _ := tree.getValue("GET", request.path)

		if handler == nil {
			if !request.nilHandler {
				t.Errorf("handle mismatch for route '%s': Expected non-nil handle", request.path)
			}
		} else if request.nilHandler {
			t.Errorf("handle mismatch for route '%s': Expected nil handle", request.path)
		} else {
			handler(nil, nil, nil)
			if fakeHandlerValue != request.route {
				t.Errorf("handle mismatch for route '%s': Wrong handle (%s != %s)", request.path, fakeHandlerValue, request.route)
			}
		}

		if !reflect.DeepEqual(vars, request.vars) {
			t.Errorf("vars mismatch for route '%s'", request.path)
		}
	}
}

func checkPriorities(t *testing.T, n *node) uint32 {
	var prio uint32
	for i := range n.children {
		prio += checkPriorities(t, n.children[i])
	}
	prio += uint32(len(n.handle))

	if n.priority != prio {
		t.Errorf(
			"priority mismatch for node '%s': is %d, should be %d",
			n.path, n.priority, prio,
		)
	}

	return prio
}

func TestTreeAddAndGet(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
	}
	for _, route := range routes {
		tree.addRoute("GET", route, fakeHandler(route))
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/a", false, "/a", nil},
		{"/", true, "", nil},
		{"/hi", false, "/hi", nil},
		{"/contact", false, "/contact", nil},
		{"/co", false, "/co", nil},
		{"/con", true, "", nil},  // key mismatch
		{"/cona", true, "", nil}, // key mismatch
		{"/no", true, "", nil},   // no matching child
		{"/ab", false, "/ab", nil},
	})

	checkPriorities(t, tree)
}

func TestTreeWildcard(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/",
		"/cmd/:tool/:sub",
		"/cmd/:tool/",
		"/src/*filepath",
		"/search/",
		"/search/:query",
		"/user_:name",
		"/user_:name/about",
		"/files/:dir/*filepath",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
	}
	for _, route := range routes {
		tree.addRoute("GET", route, fakeHandler(route))
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/cmd/test/", false, "/cmd/:tool/", map[string]string{"tool": "test"}},
		{"/cmd/test", true, "", map[string]string{"tool": "test"}},
		{"/cmd/test/3", false, "/cmd/:tool/:sub", map[string]string{"tool": "test", "sub": "3"}},
		{"/src/", false, "/src/*filepath", map[string]string{"filepath": "/"}},
		{"/src/some/file.png", false, "/src/*filepath", map[string]string{"filepath": "/some/file.png"}},
		{"/search/", false, "/search/", nil},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", map[string]string{"query": "someth!ng+in+ünìcodé"}},
		{"/search/someth!ng+in+ünìcodé/", true, "", map[string]string{"query": "someth!ng+in+ünìcodé"}},
		{"/user_gopher", false, "/user_:name", map[string]string{"name": "gopher"}},
		{"/user_gopher/about", false, "/user_:name/about", map[string]string{"name": "gopher"}},
		{"/files/js/inc/framework.js", false, "/files/:dir/*filepath", map[string]string{"dir": "js", "filepath": "/inc/framework.js"}},
	})

	checkPriorities(t, tree)
}

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}

type testRoute struct {
	path     string
	conflict bool
}

func testRoutes(t *testing.T, routes []testRoute) {
	tree := &node{}

	for _, route := range routes {
		recv := catchPanic(func() {
			tree.addRoute("GET", route.path, nil)
		})

		if route.conflict {
			if recv == nil {
				t.Errorf("no panic for conflicting route '%s'", route.path)
			}
		} else if recv != nil {
			t.Errorf("unexpected panic for route '%s': %v", route.path, recv)
		}
	}

	//printChildren(tree, "")
}

func TestTreeWildcardConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/:tool/:sub", false},
		{"/cmd/vet", true},
		{"/src/*filepath", false},
		{"/src/*filepathx", true},
		{"/src/", true},
		{"/src1/", false},
		{"/src1/*filepath", true},
		{"/src2*filepath", true},
		{"/search/:query", false},
		{"/search/invalid", true},
		{"/user_:name", false},
		{"/user_x", true},
		{"/user_:name", false},
		{"/id:id", false},
		{"/id/:id", true},
	}
	testRoutes(t, routes)
}

func TestTreeChildConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/vet", false},
		{"/cmd/:tool/:sub", true},
		{"/src/AUTHORS", false},
		{"/src/*filepath", true},
		{"/user_x", false},
		{"/user_:name", true},
		{"/id/:id", false},
		{"/id:id", true},
		{"/:id", true},
		{"/*filepath", true},
	}
	testRoutes(t, routes)
}

func TestTreeDupliatePath(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/",
		"/doc/",
		"/src/*filepath",
		"/search/:query",
		"/user_:name",
	}
	for _, route := range routes {
		recv := catchPanic(func() {
			tree.addRoute("GET", route, fakeHandler(route))
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", route, recv)
		}

		// Add again
		recv = catchPanic(func() {
			tree.addRoute("GET", route, nil)
		})
		if recv == nil {
			t.Fatalf("no panic while inserting duplicate route '%s", route)
		}
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/doc/", false, "/doc/", nil},
		{"/src/some/file.png", false, "/src/*filepath", map[string]string{"filepath": "/some/file.png"}},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", map[string]string{"query": "someth!ng+in+ünìcodé"}},
		{"/user_gopher", false, "/user_:name", map[string]string{"name": "gopher"}},
	})
}

func TestEmptyWildcardName(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/user:",
		"/user:/",
		"/cmd/:/",
		"/src/*",
	}
	for _, route := range routes {
		recv := catchPanic(func() {
			tree.addRoute("GET", route, nil)
		})
		if recv == nil {
			t.Fatalf("no panic while inserting route with empty wildcard name '%s", route)
		}
	}
}

func TestTreeCatchAllConflict(t *testing.T) {
	routes := []testRoute{
		{"/src/*filepath/x", true},
		{"/src2/", false},
		{"/src2/*filepath/x", true},
	}
	testRoutes(t, routes)
}

/*func TestTreeDuplicateWildcard(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/:id/:name/:id",
	}
	for _, route := range routes {
		...
	}
}*/

func TestTreeTrailingSlashRedirect(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/hi",
		"/b/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/no/a",
		"/no/b",
	}
	for _, route := range routes {
		recv := catchPanic(func() {
			tree.addRoute("GET", route, fakeHandler(route))
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", route, recv)
		}
	}

	//printChildren(tree, "")

	tsrRoutes := [...]string{
		"/hi/",
		"/b",
		"/search/gopher/",
		"/cmd/vet",
		"/src",
		"/x/",
		"/y",
		"/0/go/",
		"/1/go",
		"/a",
		"/doc/",
	}
	for _, route := range tsrRoutes {
		handler, _, tsr := tree.getValue("GET", route)
		if handler != nil {
			t.Fatalf("non-nil handler for TSR route '%s", route)
		} else if !tsr {
			t.Errorf("expected TSR recommendation for route '%s'", route)
		}
	}

	noTsrRoutes := [...]string{
		"/",
		"/no",
		"/no/",
		"/_",
		"/_/",
	}
	for _, route := range noTsrRoutes {
		handler, _, tsr := tree.getValue("GET", route)
		if handler != nil {
			t.Fatalf("non-nil handler for No-TSR route '%s", route)
		} else if tsr {
			t.Errorf("expected no TSR recommendation for route '%s'", route)
		}
	}
}

// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"strings"
	"unicode"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

type nodeType uint8

const (
	static   nodeType = 0
	param    nodeType = 1
	catchAll nodeType = 2
)

type node struct {
	path      string
	indices   []byte
	children  []*node
	wildChild bool
	nType     nodeType
	handle    map[string]Handle
	priority  uint32
}

// increments priority of the given child and reorders if necessary
func (n *node) incrementChildPrio(i int) int {
	n.children[i].priority++
	prio := n.children[i].priority

	// adjust position (move to front)
	for j := i - 1; j >= 0 && n.children[j].priority < prio; j-- {
		// swap node positions
		tmpN := n.children[j]
		n.children[j] = n.children[i]
		n.children[i] = tmpN
		tmpI := n.indices[j]
		n.indices[j] = n.indices[i]
		n.indices[i] = tmpI

		i--
	}
	return i
}

// addRoute adds a node with the given handle to the path.
// Not concurrency-safe!
func (n *node) addRoute(method, path string, handle Handle) {
	n.priority++
	// non-empty tree
	if len(n.path) > 0 || len(n.children) > 0 {
	WALK:
		for {
			// Find the longest common prefix.
			// This also implies that the commom prefix contains no ':' or '*'
			// since the existing key can't contain this chars.
			i := 0
			for j := min(len(path), len(n.path)); i < j && path[i] == n.path[i]; i++ {
			}

			// Split edge
			if i < len(n.path) {
				n.children = []*node{&node{
					path:      n.path[i:],
					indices:   n.indices,
					children:  n.children,
					handle:    n.handle,
					wildChild: n.wildChild,
					priority:  n.priority - 1,
				}}
				n.indices = []byte{n.path[i]}
				n.path = path[:i]
				n.handle = nil
				n.wildChild = false
			}

			// Make new node a child of this node
			if i < len(path) {
				path = path[i:]

				if n.wildChild {
					n = n.children[0]
					n.priority++

					// Check if the wildcard matches
					if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
						// check for longer wildcard, e.g. :name and :names
						if len(n.path) >= len(path) || path[len(n.path)] == '/' {
							continue WALK
						}
					}

					panic("conflict with wildcard route")
				}

				c := path[0]

				// param
				if n.nType == param && c == '/' && len(n.children) == 1 {
					n = n.children[0]
					n.priority++
					continue WALK
				}

				// Check if a child with the next path byte exists
				for i, index := range n.indices {
					if c == index {
						i = n.incrementChildPrio(i)
						n = n.children[i]
						continue WALK
					}
				}

				// Otherwise insert it
				if c != ':' && c != '*' {
					n.indices = append(n.indices, c)
					child := &node{}
					n.children = append(n.children, child)

					n.incrementChildPrio(len(n.indices) - 1)
					n = child
				}
				n.insertChild(method, path, handle)
				return

			} else if i == len(path) { // Make node a (in-path) leaf
				if n.handle == nil {
					n.handle = map[string]Handle{
						method: handle,
					}
				} else {
					if n.handle[method] != nil {
						panic("a Handle is already registered for this method at this path")
					}
					n.handle[method] = handle
				}
			}
			return
		}
	} else { // Empty tree
		n.insertChild(method, path, handle)
	}
}

func (n *node) insertChild(method, path string, handle Handle) {
	var offset int

	// find prefix until first wildcard (beginning with ':'' or '*'')
	for i, j := 0, len(path); i < j; i++ {
		if c := path[i]; c == ':' || c == '*' {
			// Check if this Node existing children which would be
			// unreachable if we insert the wildcard here
			if len(n.children) > 0 {
				panic("wildcard route conflicts with existing children")
			}

			// find wildcard end (either '/'' or path end)
			k := i + 1
			for k < j && path[k] != '/' {
				k++
			}

			if k-i == 1 {
				panic("wildcards must be named with a non-empty name")
			}

			if c == ':' { // param
				// split path at the beginning of the wildcard
				if i > 0 {
					n.path = path[offset:i]
					offset = i
				}

				child := &node{
					nType: param,
				}
				n.children = []*node{child}
				n.wildChild = true
				n = child
				n.priority++

				// if the path doesn't end with the wildcard, then there will be
				// another non-wildcard subpath starting with '/'
				if k < j {
					n.path = path[offset:k]
					offset = k

					child := &node{}
					n.children = []*node{child}
					n = child
					n.priority++
				}

			} else { // catchAll
				if len(path) != k {
					panic("catch-all routes are only allowed at the end of the path")
				}

				if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
					panic("catch-all conflicts with existing handle for the path segment root")
				}

				// currently fixed width 1 for '/'
				i--
				if path[i] != '/' {
					panic("no / before catch-all")
				}

				n.path = path[offset:i]

				// first node: catchAll node with empty path
				child := &node{
					wildChild: true,
					nType:     catchAll,
				}
				n.children = []*node{child}
				n.indices = []byte{path[i]}
				n = child
				n.priority++

				// second node: node holding the variable
				child = &node{
					path: path[i:],
					handle: map[string]Handle{
						method: handle,
					},
					nType:    catchAll,
					priority: 1,
				}
				n.children = []*node{child}

				return
			}
		}
	}

	// insert remaining path part and handle to the leaf
	n.path = path[offset:]
	n.handle = map[string]Handle{
		method: handle,
	}
}

// Returns the handle registered with the given path (key). The values of
// wildcards are saved to a map.
// If no handle can be found, a TSR (trailing slash redirect) recommendation is
// made if a handle exists with an extra (without the) trailing slash for the
// given path.
func (n *node) getValue(method, path string) (handle Handle, vars map[string]string, tsr bool) {
WALK: // Outer loop for walking the tree nodes
	for len(path) >= len(n.path) && path[:len(n.path)] == n.path {
		path = path[len(n.path):]

		if len(path) == 0 {
			// Check if this node has a handle registered for the given node
			if handle = n.handle[method]; handle != nil {
				return
			}

			// No handle found. Check if a handle for this path + a
			// trailing slash exists for trailing slash recommendation
			for i, index := range n.indices {
				if index == '/' {
					n = n.children[i]
					tsr = (n.path == "/" && n.handle[method] != nil) ||
						(n.nType == catchAll && n.children[0].handle[method] != nil)
					return
				}
			}

			// TODO: handle HTTP Error 405 - Method Not Allowed
			// Return available methods

			return

		} else if n.wildChild {
			n = n.children[0]

			switch n.nType {
			case param:
				// find param end (either '/'' or path end)
				k := 0
				for k < len(path) && path[k] != '/' {
					k++
				}

				// save param value
				if vars == nil {
					vars = map[string]string{
						n.path[1:]: path[:k],
					}
				} else {
					vars[n.path[1:]] = path[:k]
				}

				// we need to go deeper!
				if k < len(path) {
					if len(n.children) > 0 {
						path = path[k:]
						n = n.children[0]
						continue
					} else { // ... but we can't
						tsr = (len(path) == k+1)
						return
					}
				}

				if handle = n.handle[method]; handle != nil {
					return
				} else if len(n.children) == 1 {
					// No handle found. Check if a handle for this path + a
					// trailing slash exists for TSR recommendation
					n = n.children[0]
					tsr = (n.path == "/" && n.handle[method] != nil)
				}

				// TODO: handle HTTP Error 405 - Method Not Allowed
				// Return available methods

				return

			case catchAll:
				// save catchAll value
				if vars == nil {
					vars = map[string]string{
						n.path[2:]: path,
					}
				} else {
					vars[n.path[2:]] = path
				}

				handle = n.handle[method]
				return

			default:
				panic("Unknown node type")
			}

		} else {
			c := path[0]

			for i, index := range n.indices {
				if c == index {
					n = n.children[i]
					continue WALK
				}
			}

			// Nothing found. We can recommend to redirect to the same URL
			// without a trailing slash if a leaf exists for that path
			tsr = (path == "/" && n.handle[method] != nil)
			return
		}
	}

	// Nothing found. We can recommend to redirect to the same URL with an extra
	// trailing slash if a leaf exists for that path
	tsr = (len(path)+1 == len(n.path) && n.path[len(path)] == '/' && n.handle[method] != nil) || (path == "/")
	return
}

// Makes a case-insensitive lookup of the given path and tries to find a handler
// for the given request method. It can optionally also fix trailing slashes.
// It returns the case-corrected path and a bool indicating wether the lookup
// was successful.
func (n *node) findCaseInsensitivePath(method, path string, fixTrailingSlash bool) (ciPath []byte, found bool) {
	ciPath = make([]byte, 0, len(path)+1) // preallocate enough memory

	// Outer loop for walking the tree nodes
	for len(path) >= len(n.path) && strings.ToLower(path[:len(n.path)]) == strings.ToLower(n.path) {
		path = path[len(n.path):]
		ciPath = append(ciPath, n.path...)

		if len(path) == 0 {
			// Check if this node has a handle registered for the given node
			if n.handle[method] != nil {
				return ciPath, true
			}

			// No handle found.
			// Try to fix the path by adding a trailing slash
			if fixTrailingSlash {
				for i, index := range n.indices {
					if index == '/' {
						n = n.children[i]
						if (n.path == "/" && n.handle[method] != nil) ||
							(n.nType == catchAll && n.children[0].handle[method] != nil) {
							return append(ciPath, '/'), true
						}
						return
					}
				}
			}
			return

		} else if n.wildChild {
			n = n.children[0]

			switch n.nType {
			case param:
				// find param end (either '/'' or path end)
				k := 0
				for k < len(path) && path[k] != '/' {
					k++
				}

				// add param value to case insensitive path
				ciPath = append(ciPath, path[:k]...)

				// we need to go deeper!
				if k < len(path) {
					if len(n.children) > 0 {
						path = path[k:]
						n = n.children[0]
						continue
					} else { // ... but we can't
						if fixTrailingSlash && len(path) == k+1 {
							return ciPath, true
						}
						return
					}
				}

				if n.handle[method] != nil {
					return ciPath, true
				} else if fixTrailingSlash && len(n.children) == 1 {
					// No handle found. Check if a handle for this path + a
					// trailing slash exists
					n = n.children[0]
					if n.path == "/" && n.handle[method] != nil {
						return append(ciPath, '/'), true
					}
				}
				return

			case catchAll:
				return append(ciPath, path...), true

			default:
				panic("Unknown node type")
			}

		} else {
			r := unicode.ToLower(rune(path[0]))
			for i, index := range n.indices {
				// must use recursive approach since both index and
				// ToLower(index) could exist. We must check both.
				if r == unicode.ToLower(rune(index)) {
					out, found := n.children[i].findCaseInsensitivePath(method, path, fixTrailingSlash)
					if found {
						return append(ciPath, out...), true
					}
				}
			}

			// Nothing found. We can recommend to redirect to the same URL
			// without a trailing slash if a leaf exists for that path
			found = (fixTrailingSlash && path == "/" && n.handle[method] != nil)
			return
		}
	}

	// Nothing found.
	// Try to fix the path by adding / removing a trailing slash
	if fixTrailingSlash {
		if len(path)+1 == len(n.path) && n.path[len(path)] == '/' &&
			strings.ToLower(path) == strings.ToLower(n.path[:len(path)]) &&
			n.handle[method] != nil {
			return append(ciPath, n.path...), true
		} else if path == "/" {
			return ciPath, true
		}
	}
	return
}

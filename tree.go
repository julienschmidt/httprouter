// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"errors"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

var (
	ErrDuplicatePath     = errors.New("a Handle is already registered for this method at this path")
	ErrEmptyWildcardName = errors.New("wildcards must be named with a non-empty name")
	ErrCatchAllConflict  = errors.New("catchAlls are only allowed at the end of the path")
	ErrChildConflict     = errors.New("can't insert a wildcard route because this path has existing children")
	ErrWildCardConflict  = errors.New("conflict with wildcard route")
)

type node struct {
	// parent *node
	path       string
	indices    []byte
	children   []*node
	handle     map[string]Handle
	wildChild  bool
	isParam    bool
	isCatchAll bool
}

// addRoute adds a leaf with the given handle to the path.
// Attention! Not concurrency-safe!
func (n *node) addRoute(method, path string, handle Handle) error {
	// non-empty tree
	if len(n.path) != 0 {
	OUTER:
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
				}}
				n.indices = []byte{n.path[i]}
				n.path = path[:i]
				n.handle = nil
				n.wildChild = false
			}

			// Make new Node a child of this node
			if i < len(path) {
				path = path[i:]

				if n.wildChild {
					n = n.children[0]

					// Check if the wildcard matches
					if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
						// check for longer wildcard, e.g. :name and :namex
						if len(n.path) < len(path) && path[len(n.path)] != '/' {
							return ErrWildCardConflict
						}
						continue OUTER
					} else {
						return ErrWildCardConflict
					}
				}

				c := path[0]

				// TODO: allow variable delimiter
				if n.isParam && c == '/' && len(n.children) == 1 {
					n = n.children[0]
					continue OUTER
				}

				// Check if a child with the next path byte exists
				for i, index := range n.indices {
					if c == index {
						n = n.children[i]
						continue OUTER
					}
				}

				if c != ':' && c != '*' {
					n.indices = append(n.indices, c)
					child := &node{}
					n.children = append(n.children, child)

					n = child
				}
				return n.insertChild(method, path, handle)

			} else if i == len(path) { // Make node a (in-path) leaf
				if n.handle == nil {
					n.handle = map[string]Handle{
						method: handle,
					}
				} else {
					if n.handle[method] != nil {
						return ErrDuplicatePath
					}
					n.handle[method] = handle
				}
			}
			return nil
		}
	} else { // Empty tree
		return n.insertChild(method, path, handle)
	}
}

func (n *node) insertChild(method, path string, handle Handle) error {
	var offset int

	// find prefix until first wildcard (beginning with ':'' or '*'')
	for i, j := 0, len(path); i < j; i++ {
		if b := path[i]; b == ':' || b == '*' {
			// Check if this Node existing children which would be
			// unreachable if we insert the wildcard here
			if len(n.children) > 0 {
				return ErrChildConflict
			}

			// find wildcard end (either '/'' or path end)
			k := i + 1
			for k < j && path[k] != '/' {
				k++
			}

			if k-i == 1 {
				return ErrEmptyWildcardName
			}

			if b == '*' && len(path) != k {
				return ErrCatchAllConflict
			}

			// split path at the beginning of the wildcard
			child := &node{}
			if b == ':' {
				child.isParam = true
			} else {
				child.isCatchAll = true
			}

			if i > 0 {
				n.path = path[offset:i]
				offset = i
			}

			n.children = []*node{child}
			n.wildChild = true
			n = child

			// if the path doesn't end with the wildcard, then there will be
			// another non-wildcard subpath starting with '/'
			if k < j {
				n.path = path[offset:k]
				offset = k

				child := &node{}
				n.children = []*node{child}
				n = child
			}
		}
	}

	// insert remaining path part and handle to the leaf
	n.path = path[offset:]
	n.handle = map[string]Handle{
		method: handle,
	}

	return nil
}

// Returns the handle registered with the given path (key). The values of
// wildcards are saved to a map.
// If no handle can be found, a TSR (trailing slash redirect) recommendation is
// made if a handle exists with an extra (without the) trailing slash for the
// given path.
func (n *node) getValue(method, path string) (handle Handle, vars map[string]string, tsr bool) {
	// Walk tree nodes
OUTER:
	for len(path) >= len(n.path) && path[:len(n.path)] == n.path {
		path = path[len(n.path):]

		if len(path) == 0 {
			// Check if this node has a handle registered for the given node
			if handle = n.handle[method]; handle != nil {
				return
			}

			// No handle found. Check if a handle for this path + a
			// trailing slash exists for TSR recommendation
			for i, index := range n.indices {
				if index == '/' {
					n = n.children[i]
					tsr = (n.path == "/" && n.handle[method] != nil)
					return
				}
			}

			// TODO: handle HTTP Error 405 - Method Not Allowed
			// Return available methods

			return

		} else if n.wildChild == true {
			n = n.children[0]

			if n.isParam {
				// find param end (either '/'' or path end)
				k := 0
				l := len(path)
				for k < l && path[k] != '/' {
					k++
				}

				// save param handle
				if vars == nil {
					vars = map[string]string{
						n.path[1:]: path[:k],
					}
				} else {
					vars[n.path[1:]] = path[:k]
				}

				// we need to go deeper!
				if k < l {
					if len(n.children) > 0 {
						path = path[k:]
						n = n.children[0]
						continue
					} else { // ... but we can't
						tsr = (l == k+1)
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

			} else { // catchAll
				// save handle
				if vars == nil {
					vars = map[string]string{
						n.path[1:]: path,
					}
				} else {
					vars[n.path[1:]] = path
				}

				handle = n.handle[method]
				return
			}

		} else {
			c := path[0]

			for i, index := range n.indices {
				if c == index {
					n = n.children[i]
					continue OUTER
				}
			}

			// Nothing found. We can recommend to redirect to the same URL without
			// a trailing slash if a leaf exists for that path
			tsr = (path == "/" && n.handle[method] != nil)
			return
		}
	}

	// Nothing found. We can recommend to redirect to the same URL with an extra
	// trailing slash if a leaf exists for that path
	tsr = (n.handle != nil && len(path)+1 == len(n.path) && n.path[len(path)] == '/') || (path == "/")
	return
}

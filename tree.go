// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package router

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
	ErrDuplicatePath     = errors.New("Duplicate Path")
	ErrEmptyWildcardName = errors.New("Wildcards must be named with a non-empty name")
	ErrCatchAllConflict  = errors.New("CatchAlls are only allowed at the end of the path")
	ErrChildConflict     = errors.New("Can't insert a wildcard route because this path has existing children")
	ErrWildCardConflict  = errors.New("Conflict with wildcard route")
)

type node struct {
	// parent *node
	key        string
	indices    []byte
	children   []*node
	value      HandlerFunc
	wildChild  bool
	isParam    bool
	isCatchAll bool
}

// addRoute adds a leaf with the given value to the path determined by the given
// key.
// Attention! Not concurrency-safe!
func (n *node) addRoute(key string, value HandlerFunc) error {
	// non-empty tree
	if len(n.key) != 0 {
	OUTER:
		for {
			// find longest common prefix
			// this also implies that the commom prefix contains no ':' or '*'
			// since the existing key can't contain this chars
			i := 0
			for j := min(len(key), len(n.key)); i < j && key[i] == n.key[i]; i++ {
			}

			// Split edge
			if i < len(n.key) {
				n.children = []*node{&node{
					key:       n.key[i:],
					indices:   n.indices,
					children:  n.children,
					value:     n.value,
					wildChild: n.wildChild,
				}}
				n.indices = []byte{n.key[i]}
				n.key = key[:i]
				n.value = nil
				n.wildChild = false
			}

			// Make new Node a child of this node
			if i < len(key) {
				key = key[i:]

				if n.wildChild {
					n = n.children[0]

					// Check if the wildcard matches
					if len(key) >= len(n.key) && n.key == key[:len(n.key)] {
						// check for longer wildcard, e.g. :name and :namex
						if len(n.key) < len(key) && key[len(n.key)] != '/' {
							return ErrWildCardConflict
						}
						continue OUTER
					} else {
						return ErrWildCardConflict
					}
				}

				c := key[0]

				// TODO: remove / edit for variable delimiter
				if n.isParam && c == '/' && len(n.children) == 1 {
					n = n.children[0]
					continue OUTER
				}

				// Check if a child with the next key byte exists
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
				return n.insertRoute(key, value)

			} else if i == len(key) { // Make node a (in-path) leaf
				if n.value != nil {
					return ErrDuplicatePath
				}
				n.value = value
			}
			return nil
		}
	} else { // Empty tree
		return n.insertRoute(key, value)
	}
}

func (n *node) insertRoute(key string, value HandlerFunc) error {
	var offset int

	// find prefix until first wildcard (beginning with ':'' or '*'')
	for i, j := 0, len(key); i < j; i++ {
		if b := key[i]; b == ':' || b == '*' {
			// Check if this Node existing children which would be
			// unreachable if we insert the wildcard here
			if len(n.children) > 0 {
				return ErrChildConflict
			}

			// find wildcard end (either '/'' or key end)
			k := i + 1
			for k < j && key[k] != '/' {
				k++
			}

			if k-i == 1 {
				return ErrEmptyWildcardName
			}

			if b == '*' && len(key) != k {
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
				n.key = key[offset:i]
				offset = i
			}

			n.children = []*node{child}
			n.wildChild = true
			n = child

			// if the path doesn't end with the wildcard, then there will be
			// another non-wildcard subpath starting with '/'
			if k < j {
				n.key = key[offset:k]
				offset = k

				child := &node{}
				n.children = []*node{child}
				n = child
			}
		}
	}

	// insert remaining key part and value to the leaf
	n.key = key[offset:]
	n.value = value

	return nil
}

// Returns the handler registered with the given path (key). The values of
// wildcards are saved to a map.
// If no handler can be found, a TSR (trailing slash redirect) recommendation is
// made if a handler exists with an extra (without the) trailing slash for the
// given path.
func (n *node) getValue(key string) (value HandlerFunc, vars map[string]string, tsr bool) {
	// Walk tree nodes
OUTER:
	for len(key) >= len(n.key) && key[:len(n.key)] == n.key {

		if len(key) == len(n.key) {
			// Check if this node has a registered handler
			if value = n.value; value != nil {
				return
			}

			// No handler found. Check if a handler for this path + a
			// trailing slash exists for TSR recommendation
			for i, index := range n.indices {
				if index == '/' {
					n = n.children[i]
					tsr = (n.key == "/" && n.value != nil)
					return
				}
			}
			return

		} else if n.wildChild == true {
			key = key[len(n.key):]
			n = n.children[0]

			if n.isParam {
				// find param end (either '/'' or key end)
				k := 0
				l := len(key)
				for k < l && key[k] != '/' {
					k++
				}

				// save param value
				if vars == nil {
					//vars = new(Vars)
					//vars.add(n.key[1:], key[:k])
					vars = map[string]string{
						n.key[1:]: key[:k],
					}
				} else {
					vars[n.key[1:]] = key[:k]
				}

				// we need to go deeper!
				if k < l {
					if len(n.children) > 0 {
						key = key[k:]
						n = n.children[0]
						continue
					} else { // ... but we can't
						tsr = (l == k+1)
						return
					}
				}

				if value = n.value; value != nil {
					return
				} else if len(n.children) == 1 {
					// No handler found. Check if a handler for this path + a
					// trailing slash exists for TSR recommendation
					n = n.children[0]
					tsr = (n.key == "/" && n.value != nil)
				}
				return

			} else { // catchAll
				// save value
				if vars == nil {
					vars = map[string]string{
						n.key[1:]: key,
					}
				} else {
					vars[n.key[1:]] = key
				}

				value = n.value
				return
			}

		} else {
			key = key[len(n.key):]
			c := key[0]

			for i, index := range n.indices {
				if c == index {
					n = n.children[i]
					continue OUTER
				}
			}

			// Nothing found. We can recommend to redirect to the same URL without
			// a trailing slash if a leaf exists for that path
			tsr = (key == "/" && n.value != nil)
			return
		}
	}

	// Nothing found. We can recommend to redirect to the same URL with an extra
	// trailing slash if a leaf exists for that path
	tsr = (n.value != nil && len(key)+1 == len(n.key) && n.key[len(key)] == '/') || (key == "/")
	return
}

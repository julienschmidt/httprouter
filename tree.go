// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

const maxParamCount uint8 = ^uint8(0)

func countParams(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != ':' && path[i] != '*' {
			continue
		}
		n++
	}
	if n >= uint(maxParamCount) {
		return maxParamCount
	}

	return uint8(n)
}

type nodeType uint8

const (
	static nodeType = iota // default
	root
	param
	catchAll
)

type node struct {
	pfx       string // first len(children) byte are indices, rest is prefix
	children  []node
	handle    Handle
	wildChild bool
	nType     nodeType
	maxParams uint8
	priority  uint32
}

// increments priority of the given child and reorders if necessary
func (n *node) incrementChildPrio(pos int) int {
	n.children[pos].priority++
	prio := n.children[pos].priority

	// adjust position (move to front)
	newPos := pos
	for newPos > 0 && n.children[newPos-1].priority < prio {
		// swap node positions
		n.children[newPos-1], n.children[newPos] = n.children[newPos], n.children[newPos-1]

		newPos--
	}

	// build new index char string
	if newPos != pos {
		n.pfx = n.pfx[:newPos] + // unchanged prefix, might be empty
			n.pfx[pos:pos+1] + // the index char we move
			n.pfx[newPos:pos] + n.pfx[pos+1:] // rest without char at 'pos'
	}

	return newPos
}

// addRoute adds a node with the given handle to the path.
// Not concurrency-safe!
func (n *node) addRoute(path string, handle Handle) {
	fullPath := path
	n.priority++
	numParams := countParams(path)

	// non-empty tree
	if len(n.pfx) > 0 {
		prefix := n.pfx[len(n.children):]
	walk:
		for {
			// Update maxParams of the current node
			if numParams > n.maxParams {
				n.maxParams = numParams
			}

			// Find the longest common prefix.
			// This also implies that the common prefix contains no ':' or '*'
			// since the existing key can't contain those chars.
			i := 0
			max := min(len(path), len(prefix))
			for i < max && path[i] == prefix[i] {
				i++
			}

			// Split edge
			if i < len(prefix) {
				child := node{
					pfx:       n.pfx[:len(n.children)] + prefix[i:],
					wildChild: n.wildChild,
					nType:     static,
					children:  n.children,
					handle:    n.handle,
					priority:  n.priority - 1,
				}

				// Update maxParams (max of all children)
				for i := range child.children {
					if child.children[i].maxParams > child.maxParams {
						child.maxParams = child.children[i].maxParams
					}
				}

				n.children = []node{child}
				// []byte for proper unicode char conversion, see #65
				n.pfx = string([]byte{prefix[i]}) + path[:i]
				n.handle = nil
				n.wildChild = false
			}

			// Make new node a child of this node
			if i < len(path) {
				path = path[i:]

				if n.wildChild {
					n = &n.children[0]
					prefix = n.pfx[len(n.children):]
					n.priority++

					// Update maxParams of the child node
					if numParams > n.maxParams {
						n.maxParams = numParams
					}
					numParams--

					// Check if the wildcard matches
					if len(path) >= len(prefix) && prefix == path[:len(prefix)] &&
						// Check for longer wildcard, e.g. :name and :names
						(len(prefix) >= len(path) || path[len(prefix)] == '/') {
						continue walk
					} else {
						// Wildcard conflict
						var pathSeg string
						if n.nType == catchAll {
							pathSeg = path
						} else {
							pathSeg = strings.SplitN(path, "/", 2)[0]
						}
						exPrefix := fullPath[:strings.Index(fullPath, pathSeg)] + prefix
						panic("'" + pathSeg +
							"' in new path '" + fullPath +
							"' conflicts with existing wildcard '" + prefix +
							"' in existing prefix '" + exPrefix +
							"'")
					}
				}

				c := path[0]

				// slash after param
				if n.nType == param && c == '/' && len(n.children) == 1 {
					n = &n.children[0]
					prefix = n.pfx[len(n.children):]
					n.priority++
					continue walk
				}

				// Check if a child with the next path byte exists
				for i := 0; i < len(n.children); i++ {
					if c == n.pfx[i] {
						i = n.incrementChildPrio(i)
						n = &n.children[i]
						prefix = n.pfx[len(n.children):]
						continue walk
					}
				}

				// Otherwise insert it
				if c != ':' && c != '*' {
					// []byte for proper unicode char conversion, see #65
					n.pfx = n.pfx[:len(n.children)] + string([]byte{c}) + n.pfx[len(n.children):]
					n.children = append(n.children, node{
						maxParams: numParams,
					})
					child := &n.children[len(n.children)-1]
					n.incrementChildPrio(len(n.children) - 1)
					n = child
				}
				n.insertChild(numParams, path, fullPath, handle)
				return

			} else if i == len(path) { // Make node a (in-path) leaf
				if n.handle != nil {
					panic("a handle is already registered for path '" + fullPath + "'")
				}
				n.handle = handle
			}
			return
		}
	} else { // Empty tree
		n.insertChild(numParams, path, fullPath, handle)
		n.nType = root
	}
}

func (n *node) insertChild(numParams uint8, path, fullPath string, handle Handle) {
	var offset int // already handled bytes of the path

	// find prefix until first wildcard (beginning with ':'' or '*'')
	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != ':' && c != '*' {
			continue
		}

		// find wildcard end (either '/' or path end)
		end := i + 1
		for end < max && path[end] != '/' {
			switch path[end] {
			// the wildcard name must not contain ':' and '*'
			case ':', '*':
				panic("only one wildcard per path segment is allowed, has: '" +
					path[i:] + "' in path '" + fullPath + "'")
			default:
				end++
			}
		}

		// check if this Node existing children which would be
		// unreachable if we insert the wildcard here
		if len(n.children) > 0 {
			panic("wildcard route '" + path[i:end] +
				"' conflicts with existing children in path '" + fullPath + "'")
		}

		// check if the wildcard has a name
		if end-i < 2 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		if c == ':' { // param
			// split path at the beginning of the wildcard
			if i > 0 {
				n.pfx = ":" + path[offset:i]
				offset = i
			} else {
				n.pfx = ":" + n.pfx
			}

			n.children = []node{node{
				nType:     param,
				maxParams: numParams,
			}}
			n.wildChild = true
			n = &n.children[0]
			n.priority++
			numParams--

			// if the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			if end < max {
				n.pfx = "/" + path[offset:end]
				offset = end

				n.children = []node{node{
					maxParams: numParams,
					priority:  1,
				}}
				n = &n.children[0]
			}

		} else { // catchAll
			if end != max || numParams > 1 {
				panic("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
			}

			if len(n.pfx)-len(n.children) > 0 && n.pfx[len(n.pfx)-1] == '/' {
				panic("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
			}

			// currently fixed width 1 for '/'
			i--
			if path[i] != '/' {
				panic("no / before catch-all in path '" + fullPath + "'")
			}

			n.pfx = string(path[i]) + path[offset:i]

			// first node: catchAll node with empty path
			n.children = []node{node{
				pfx:       "/",
				wildChild: true,
				nType:     catchAll,
				maxParams: 1,
			}}
			n = &n.children[0]
			n.priority++

			// second node: node holding the variable
			n.children = []node{node{
				pfx:       path[i:],
				nType:     catchAll,
				maxParams: 1,
				handle:    handle,
				priority:  1,
			}}

			return
		}
	}

	// insert remaining path part and handle to the leaf
	n.pfx = n.pfx[:len(n.children)] + path[offset:]
	n.handle = handle
}

// Returns the handle registered with the given path (key). The values of
// wildcards are saved to a map.
// If no handle can be found, a TSR (trailing slash redirect) recommendation is
// made if a handle exists with an extra (without the) trailing slash for the
// given path.
func (n node) getValue(path string) (handle Handle, p Params, tsr bool) {
walk: // outer loop for walking the tree
	for {
		prefix := n.pfx[len(n.children):]

		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]
				// If this node does not have a wildcard (param or catchAll)
				// child, we can just look up the next child node and continue
				// to walk down the tree
				if !n.wildChild {
					for i, max, c := 0, len(n.children), path[0]; i < max; i++ {
						if c == n.pfx[i] {
							n = n.children[i]
							continue walk
						}
					}

					// Nothing found.
					// We can recommend to redirect to the same URL without a
					// trailing slash if a leaf exists for that path.
					tsr = (path == "/" && n.handle != nil)
					return

				}

				// handle wildcard child
				n = n.children[0]
				switch n.nType {
				case param:
					// find param end (either '/' or path end)
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					// save param value
					if p == nil {
						// lazy allocation
						p = make(Params, 0, n.maxParams)
					}
					i := len(p)
					p = p[:i+1] // expand slice within preallocated capacity
					p[i].Key = n.pfx[len(n.children)+1:]
					p[i].Value = path[:end]

					// we need to go deeper!
					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}

						// ... but we can't
						tsr = (len(path) == end+1)
						return
					}

					if handle = n.handle; handle != nil {
						return
					} else if len(n.children) == 1 {
						// No handle found. Check if a handle for this path + a
						// trailing slash exists for TSR recommendation
						n = n.children[0]
						tsr = (n.handle != nil && n.pfx[len(n.children):] == "/")
					}

					return

				case catchAll:
					// save param value
					if p == nil {
						// lazy allocation
						p = make(Params, 0, n.maxParams)
					}
					i := len(p)
					p = p[:i+1]          // expand slice within preallocated capacity
					p[i].Key = n.pfx[2:] // TODO?
					p[i].Value = path

					handle = n.handle
					return

				default:
					panic("invalid node type")
				}
			}
		} else if path == prefix {
			// We should have reached the node containing the handle.
			// Check if this node has a handle registered.
			if handle = n.handle; handle != nil {
				return
			}

			if path == "/" && n.wildChild && n.nType != root {
				tsr = true
				return
			}

			// No handle found. Check if a handle for this path + a
			// trailing slash exists for trailing slash recommendation
			for i, max := 0, len(n.children); i < max; i++ {
				if n.pfx[i] == '/' {
					n = n.children[i]
					tsr = (len(n.pfx)-len(n.children) == 1 && n.handle != nil) ||
						(n.nType == catchAll && n.children[0].handle != nil)
					return
				}
			}

			return
		}

		// Nothing found. We can recommend to redirect to the same URL with an
		// extra trailing slash if a leaf exists for that path
		tsr = (path == "/") ||
			(len(prefix) == len(path)+1 && prefix[len(path)] == '/' &&
				path == prefix[:len(prefix)-1] && n.handle != nil)
		return
	}
}

// Makes a case-insensitive lookup of the given path and tries to find a handler.
// It can optionally also fix trailing slashes.
// It returns the case-corrected path and a bool indicating whether the lookup
// was successful.
func (n *node) findCaseInsensitivePath(path string, fixTrailingSlash bool) (ciPath []byte, found bool) {
	return n.findCaseInsensitivePathRec(
		path,
		make([]byte, 0, len(path)+1), // preallocate enough memory for new path
		[4]byte{},                    // empty rune buffer
		fixTrailingSlash,
	)
}

// shift bytes in array by n bytes left
func shiftNRuneBytes(rb [4]byte, n int) [4]byte {
	switch n {
	case 0:
		return rb
	case 1:
		return [4]byte{rb[1], rb[2], rb[3], 0}
	case 2:
		return [4]byte{rb[2], rb[3]}
	case 3:
		return [4]byte{rb[3]}
	default:
		return [4]byte{}
	}
}

// recursive case-insensitive lookup function used by n.findCaseInsensitivePath
func (n *node) findCaseInsensitivePathRec(path string, ciPath []byte, rb [4]byte, fixTrailingSlash bool) ([]byte, bool) {

	prefix := n.pfx[len(n.children):]

walk: // outer loop for walking the tree
	for len(path) >= len(prefix) && (len(prefix) == 0 || strings.EqualFold(path[1:len(prefix)], prefix[1:])) {
		// add common prefix to result

		oldPath := path
		path = path[len(prefix):]
		ciPath = append(ciPath, prefix...)

		if len(path) > 0 {
			// If this node does not have a wildcard (param or catchAll) child,
			// we can just look up the next child node and continue to walk down
			// the tree
			if !n.wildChild {
				// skip rune bytes already processed
				rb = shiftNRuneBytes(rb, len(prefix))

				if rb[0] != 0 {
					// old rune not finished
					for i, max := 0, len(n.children); i < max; i++ {
						if n.pfx[i] == rb[0] {
							// continue with child node
							n = &n.children[i]
							prefix = n.pfx[len(n.children):]
							continue walk
						}
					}
				} else {
					// process a new rune
					var rv rune

					// find rune start
					// runes are up to 4 byte long,
					// -4 would definitely be another rune
					var off int
					for max := min(len(prefix), 3); off < max; off++ {
						if i := len(prefix) - off; utf8.RuneStart(oldPath[i]) {
							// read rune from cached path
							rv, _ = utf8.DecodeRuneInString(oldPath[i:])
							break
						}
					}

					// calculate lowercase bytes of current rune
					lo := unicode.ToLower(rv)
					utf8.EncodeRune(rb[:], lo)

					// skip already processed bytes
					rb = shiftNRuneBytes(rb, off)

					for i, max, c := 0, len(n.children), rb[0]; i < max; i++ {
						// lowercase matches
						if n.pfx[i] == c {
							// must use a recursive approach since both the
							// uppercase byte and the lowercase byte might exist
							// as an index
							if out, found := n.children[i].findCaseInsensitivePathRec(
								path, ciPath, rb, fixTrailingSlash,
							); found {
								return out, true
							}
							break
						}
					}

					// if we found no match, the same for the uppercase rune,
					// if it differs
					if up := unicode.ToUpper(rv); up != lo {
						utf8.EncodeRune(rb[:], up)
						rb = shiftNRuneBytes(rb, off)

						for i, max, c := 0, len(n.children), rb[0]; i < max; i++ {
							// uppercase matches
							if n.pfx[i] == c {
								// continue with child node
								n = &n.children[i]
								prefix = n.pfx[len(n.children):]
								continue walk
							}
						}
					}
				}

				// Nothing found. We can recommend to redirect to the same URL
				// without a trailing slash if a leaf exists for that path
				return ciPath, (fixTrailingSlash && path == "/" && n.handle != nil)
			}

			n = &n.children[0]
			switch n.nType {
			case param:
				// find param end (either '/' or path end)
				k := 0
				for k < len(path) && path[k] != '/' {
					k++
				}

				// add param value to case insensitive path
				ciPath = append(ciPath, path[:k]...)

				// we need to go deeper!
				if k < len(path) {
					if len(n.children) > 0 {
						// continue with child node
						n = &n.children[0]
						prefix = n.pfx[len(n.children):]
						path = path[k:]
						continue
					}

					// ... but we can't
					if fixTrailingSlash && len(path) == k+1 {
						return ciPath, true
					}
					return ciPath, false
				}

				if n.handle != nil {
					return ciPath, true
				} else if fixTrailingSlash && len(n.children) == 1 {
					// No handle found. Check if a handle for this path + a
					// trailing slash exists
					n = &n.children[0]
					if n.pfx[len(n.children):] == "/" && n.handle != nil {
						return append(ciPath, '/'), true
					}
				}
				return ciPath, false

			case catchAll:
				return append(ciPath, path...), true

			default:
				panic("invalid node type")
			}
		} else {
			// We should have reached the node containing the handle.
			// Check if this node has a handle registered.
			if n.handle != nil {
				return ciPath, true
			}

			// No handle found.
			// Try to fix the path by adding a trailing slash
			if fixTrailingSlash {
				for i, max := 0, len(n.children); i < max; i++ {
					if n.pfx[i] == '/' {
						n = &n.children[i]
						if (len(n.pfx)-len(n.children) == 1 && n.handle != nil) ||
							(n.nType == catchAll && n.children[0].handle != nil) {
							return append(ciPath, '/'), true
						}
						return ciPath, false
					}
				}
			}
			return ciPath, false
		}
	}

	// Nothing found.
	// Try to fix the path by adding / removing a trailing slash
	if fixTrailingSlash {
		if path == "/" {
			return ciPath, true
		}
		if len(path)+1 == len(prefix) && prefix[len(path)] == '/' &&
			strings.EqualFold(path[1:], prefix[1:len(path)]) && n.handle != nil {
			return append(ciPath, prefix...), true
		}
	}
	return ciPath, false
}

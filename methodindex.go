// Copyright 2019 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"net/http"
)

const (
	idxGET = iota
	idxHEAD
	idxPOST
	idxPUT
	idxPATCH
	idxDELETE
	idxCONNECT
	idxOPTIONS
	idxTRACE
	idxEnd
)

type customMethod struct {
	method string
	root   *node
}

type methodIndex struct {
	roots  [idxEnd]*node
	custom []customMethod
}

func (m *methodIndex) init(method string) (root *node) {
	var idx int
	switch method {
	case http.MethodGet:
		idx = idxGET
	case http.MethodPatch:
		idx = idxPATCH
	case http.MethodPost:
		idx = idxPOST
	case http.MethodPut:
		idx = idxPUT
	case http.MethodDelete:
		idx = idxDELETE
	case http.MethodHead:
		idx = idxHEAD
	case http.MethodOptions:
		idx = idxOPTIONS
	case http.MethodConnect:
		idx = idxCONNECT
	case http.MethodTrace:
		idx = idxTRACE
	default: // custom methods
		for _, c := range m.custom {
			if c.method == method {
				return c.root
			}
		}

		root = new(node)
		m.custom = append(m.custom, customMethod{method, root})
		return root
	}

	root = m.roots[idx]
	if root == nil {
		root = new(node)
		m.roots[idx] = root
	}
	return root
}

func (m *methodIndex) get(method string) *node {
	switch method[0] {
	case 'G':
		if method == http.MethodGet {
			return m.roots[idxGET]
		}
	case 'P':
		switch len(method) {
		case 5:
			if method == http.MethodPatch {
				return m.roots[idxPATCH]
			}
		case 4:
			if method == http.MethodPost {
				return m.roots[idxPOST]
			}
		case 3:
			if method == http.MethodPut {
				return m.roots[idxPUT]
			}
		}
	case 'D':
		if method == http.MethodDelete {
			return m.roots[idxDELETE]
		}
	case 'H':
		if method == http.MethodHead {
			return m.roots[idxHEAD]
		}
	case 'O':
		if method == http.MethodOptions {
			return m.roots[idxOPTIONS]
		}
	case 'C':
		if method == http.MethodConnect {
			return m.roots[idxCONNECT]
		}
	case 'T':
		if method == http.MethodTrace {
			return m.roots[idxTRACE]
		}
	}

	// custom methods
	for _, c := range m.custom {
		if c.method == method {
			return c.root
		}
	}
	return nil
}

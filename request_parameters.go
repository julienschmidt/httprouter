package httprouter

import (
	"io"
	"net/http"
)

// Parameters returns all path parameters for given
// request.
//
// If there were no parameters and route is static
// then nil is returned.
func Parameters(req *http.Request) Params {
	if req == nil {
		return nil
	}
	return parameterized(req).get()
}

type paramReadCloser interface {
	io.ReadCloser
	get() Params
	set(Params)
}

type parameters struct {
	io.ReadCloser
	all Params
}

func (p *parameters) get() Params {
	return p.all
}

func (p *parameters) set(params Params) {
	p.all = params
}

func parameterized(req *http.Request) paramReadCloser {
	p, ok := req.Body.(paramReadCloser)
	if !ok {
		p = &parameters{ReadCloser: req.Body}
		req.Body = p
	}
	return p
}

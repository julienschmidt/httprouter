// +build go1.5

package httprouter

import (
	"net/http"
)

func getPath(r *http.Request) string {
	return r.URL.EscapedPath()
}

func addTrailingSlash(r *http.Request) {
	r.URL.RawPath += "/"
	r.URL.Path += "/"
}

func removeTrailingSlash(r *http.Request) {
	rawPath := r.URL.EscapedPath()
	r.URL.RawPath = rawPath[:len(rawPath)-1]
	r.URL.Path = r.URL.Path[:len(r.URL.Path)-1]
}

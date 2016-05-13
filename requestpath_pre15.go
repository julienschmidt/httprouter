// +build !go1.5

package httprouter

import (
	"net/http"
	"strings"
)

// BUG(): In go1.4 and earlier, percent-encoded slashes are only handled
// correctly for server requests.  The http.NewRequest() function used to
// create client requests does not populate the RequestURI field.

func getPath(r *http.Request) string {
	if r.RequestURI != "" {
		return strings.SplitN(r.RequestURI, "?", 2)[0]
	}
	return r.URL.Path
}

func addTrailingSlash(r *http.Request) {
	r.URL.Path = r.URL.Path + "/"
}

func removeTrailingSlash(r *http.Request) {
	r.URL.Path = r.URL.Path[:len(r.URL.Path)-1]
}

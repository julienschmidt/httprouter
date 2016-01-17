package httprouter

import (
	"io"
	"net/http"
	"testing"

	original "github.com/julienschmidt/httprouter"
)

func BenchmarkHttprouter_Static(b *testing.B) {
	router := original.New()
	router.GET("/static/route", func(_ http.ResponseWriter, _ *http.Request, _ original.Params) {})

	benchRequest(b, router)
}

func BenchmarkVanilla_Static(b *testing.B) {
	router := New()
	router.GET("/static/route", func(_ http.ResponseWriter, _ *http.Request) {})

	benchRequest(b, router)
}

func BenchmarkHttprouter_Param(b *testing.B) {
	router := original.New()
	router.GET("/user/:name", func(w http.ResponseWriter, r *http.Request, ps original.Params) {
		io.WriteString(w, ps.ByName("name"))
	})

	benchRequest(b, router)
}

func BenchmarkVanilla_Param(b *testing.B) {
	router := New()
	router.GET("/user/:name", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, Parameters(r).ByName("name"))
	})

	benchRequest(b, router)
}

func benchRequest(b *testing.B, router http.Handler) {
	r, _ := http.NewRequest("GET", "/user/gordon", nil)
	w := new(mockResponseWriter)
	u := r.URL
	rq := u.RawQuery
	r.RequestURI = u.RequestURI()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		u.RawQuery = rq
		router.ServeHTTP(w, r)
	}
}

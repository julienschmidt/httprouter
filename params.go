package httprouter

import (
	"context"
	"net/http"
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for i := range ps {
		if ps[i].Key == name {
			return ps[i].Value
		}
	}
	return ""
}

type paramsContextKey struct{}

// WithParams returns a new Context that contains ps.
func WithParams(ctx context.Context, ps Params) context.Context {
	return context.WithValue(ctx, paramsContextKey{}, ps)
}

// GetParams returns the Params from the Context of the Request.
func GetParams(req *http.Request) Params {
	return GetParamsFromContext(req.Context())
}

// GetParamsFromContext returns the Params from the Context.
func GetParamsFromContext(ctx context.Context) Params {
	ps, _ := ctx.Value(paramsContextKey{}).(Params)
	return ps
}

// GetParam returns the value of the first Param which key matches in the Context
// of the Request.
func GetParam(req *http.Request, key string) string {
	return GetParamFromContext(req.Context(), key)
}

// GetParam returns the value of the first Param which key matches in the Context.
func GetParamFromContext(ctx context.Context, key string) string {
	return GetParamsFromContext(ctx).ByName(key)
}

package httprouter

import (
	"golang.org/x/net/context"
	"net/http"
)

// NewContext contains a function that is used to create a
// new context.Context for each request. The default implementation
// returns context.Background(), but the calling program can override
// this to return something different. For example if the calling
// program wanted to enforce a timeout for every request, it could
// override this variable with a different implemenation.
//
// Example:
//
//     func init() {
//         httprouter.NewContext = func() context.Context {
//             ctx, _ := context.WithTimeout(context.Background(), time.Second * 30)
//             return ctx
//	       }
//	   }
var NewContext func() context.Context

// backgroundContext is used as the parent context for all requests.
// This is allocated at program startup to avoid additional memory
// allocations.
var backgroundContext context.Context = context.Background()

func init() {
	NewContext = func() context.Context {
		return backgroundContext
	}
}

// contextKey is a type used for storing values in the context
type contextKey int

const (
	keyParams contextKey = iota
)

// Parameters returns the array of Param objects associated with the
// context. If there are no parameters associated with the context, an
// empty array is returned.
func Parameters(ctx context.Context) Params {
	params, ok := ctx.Value(keyParams).(Params)
	if !ok {
		params = make(Params, 0)
	}
	return params
}

// newContextWithCancel creates a new context object to be associated with the http request.
// Note that it is important that the caller arranges for cancelFunc to be called
// at the end of the request, or else there will be a goroutine leak.
func newContextWithCancel(parent context.Context, w http.ResponseWriter, r *http.Request) (context.Context, func()) {
	// create a context with a cancel function, and attach it to
	// the close notify channel if it exists.
	ctx, cancelFunc := context.WithCancel(parent)

	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		go func() {
			select {
			case <-closeNotifier.CloseNotify():
				cancelFunc()
				break
			case <-ctx.Done():
				break
			}
		}()
	}

	return ctx, cancelFunc
}

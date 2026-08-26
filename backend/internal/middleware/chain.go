package middleware

import "net/http"

// Middleware wraps an http.Handler with additional behaviour.
type Middleware func(http.Handler) http.Handler

// Chain composes middlewares in the order they are listed, i.e.
//
//	Chain(A, B, C)(handler) == A(B(C(handler)))
//
// so the first middleware in the list is the outermost (runs first on the
// way in, last on the way out). This lets route definitions read top to
// bottom in execution order:
//
//	middleware.Chain(
//	    auth.RequireAuth,
//	    auth.WithUser(userService),
//	    auth.RequireRole("PATIENT"),
//	)(http.HandlerFunc(handler))
func Chain(mws ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			final = mws[i](final)
		}
		return final
	}
}

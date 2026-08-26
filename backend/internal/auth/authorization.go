package auth

import (
	"net/http"
)

// RequireRole restricts access to callers whose AuthUser.Role is one of the
// given roles. It MUST run after RequireAuth and WithUser in the middleware
// chain, since it reads the AuthUser placed on the context by WithUser -
// it never trusts a role that hasn't been loaded from our own database.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authUser, ok := GetAuthUser(r)
			if !ok {
				// WithUser was not wired before this middleware - fail closed.
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if _, ok := allowed[authUser.Role]; !ok {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

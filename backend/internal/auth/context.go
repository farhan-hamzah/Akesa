package auth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// ErrUserNotFound is returned by a UserLoader when the authenticated Clerk
// principal has no corresponding row in our own users table yet (i.e. the
// client has not called /api/v1/users/sync after signing in with Clerk).
var ErrUserNotFound = errors.New("auth: user not found")

// AuthUser is the minimal, request-scoped identity we need for
// authorization decisions. It intentionally does not live in the `user`
// package so that `auth` never has to import `user` (which itself imports
// `auth` for GetClerkUserID). Any package that needs the full domain
// User can look it up again by ID inside its own service.
type AuthUser struct {
	ID     uuid.UUID
	Role   string
	Status string
}

// UserLoader resolves a Clerk subject (user id) to our internal AuthUser.
// The `user` package's Service implements this interface - it is defined
// here, on the consumer side, per Go convention.
type UserLoader interface {
	LoadAuthUser(ctx context.Context, clerkUserID string) (*AuthUser, error)
}

type contextKey string

const authUserContextKey contextKey = "auth.authUser"

// WithUser resolves the currently authenticated Clerk principal into our
// internal AuthUser (id, role, status) and stores it on the request
// context. It MUST run after RequireAuth. Any route that needs
// RequireRole must also run WithUser first.
func WithUser(loader UserLoader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clerkUserID, ok := GetClerkUserID(r)
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			authUser, err := loader.LoadAuthUser(r.Context(), clerkUserID)
			if err != nil {
				log.Printf(
					"WithUser: failed to load user clerk_id=%s: %v",
					clerkUserID,
					err,
				)

				if errors.Is(err, ErrUserNotFound) {
					http.Error(
						w,
						`{"error":"user_not_registered","message":"call /api/v1/users/sync first"}`,
						http.StatusForbidden,
					)
					return
				}

				http.Error(
					w,
					`{"error":"internal_server_error"}`,
					http.StatusInternalServerError,
				)
				return
			}

			if authUser.Status != "ACTIVE" {
				http.Error(w, `{"error":"account_inactive"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), authUserContextKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuthUser reads the AuthUser previously stored by WithUser.
func GetAuthUser(r *http.Request) (*AuthUser, bool) {
	u, ok := r.Context().Value(authUserContextKey).(*AuthUser)
	return u, ok
}

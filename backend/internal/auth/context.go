package auth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
)

var ErrUserNotFound = errors.New("auth: user not found")

type AuthUser struct {
	ID     uuid.UUID
	Role   string
	Status string
}

type UserLoader interface {
	LoadAuthUser(ctx context.Context, clerkUserID string) (*AuthUser, error)
}

type contextKey string

const authUserContextKey contextKey = "auth.authUser"

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

func GetAuthUser(r *http.Request) (*AuthUser, bool) {
	u, ok := r.Context().Value(authUserContextKey).(*AuthUser)
	return u, ok
}

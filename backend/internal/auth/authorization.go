package auth

import (
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/response"
)

type RoleResolver func(
	r *http.Request,
) (string, error)

func RequireRole(resolveRole RoleResolver, roles ...string) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				_, ok := GetClerkUserID(r)

				if !ok {
					response.Error(
						w,
						http.StatusUnauthorized,
						"unauthorized",
					)
					return
				}

				currentRole, err := resolveRole(r)

				if err != nil {
					response.Error(
						w,
						http.StatusUnauthorized,
						"user_not_found",
					)
					return
				}

				for _, role := range roles {
					if currentRole == role {
						next.ServeHTTP(w, r)
						return
					}
				}

				response.Error(
					w,
					http.StatusForbidden,
					"forbidden",
				)
			},
		)
	}
}

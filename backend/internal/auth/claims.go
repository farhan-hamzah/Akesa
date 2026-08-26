package auth

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
)

func GetSessionClaims(r *http.Request) (*clerk.SessionClaims, bool) {
	return clerk.SessionClaimsFromContext(r.Context())
}

func GetClerkUserID(r *http.Request) (string, bool) {
	claims, ok := GetSessionClaims(r)

	if !ok || claims.Subject == "" {
		return "", false
	}

	return claims.Subject, true
}

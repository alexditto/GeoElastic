// Package middleware holds cross-cutting http.Handler wrappers — currently
// just authentication — rather than logic specific to one resource, which
// belongs in internal/handler instead.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"geoelastic/internal/service"
)

type contextKey int

const userIDContextKey contextKey = iota

// UserIDFromContext returns the authenticated caller's user ID, as attached
// by RequireAuth. It's only meaningful inside a handler wrapped by
// RequireAuth; ok is false otherwise.
func UserIDFromContext(ctx context.Context) (id string, ok bool) {
	id, ok = ctx.Value(userIDContextKey).(string)
	return id, ok
}

// RequireAuth wraps next so it only runs for requests carrying a valid
// "Authorization: Bearer <token>" header, rejecting everything else with
// 401. On success, the token's owning user ID is attached to the request
// context for next to read via UserIDFromContext.
func RequireAuth(auth *service.Authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			unauthorized(w, "missing bearer token")
			return
		}

		userID, err := auth.Authenticate(r.Context(), token)
		if err != nil {
			unauthorized(w, "invalid or expired token")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDContextKey, userID)))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}

func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

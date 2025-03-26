package transport

import (
	"context"
	"net/http"

	"github.com/TejaswinSingh/version0/services/governance/auth"
	governanceErrors "github.com/TejaswinSingh/version0/services/governance/errors"
)

type contextKey string

const claimsKey contextKey = "claims"

func JWTAuthMiddleware(secretKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			encodeError(r.Context(), governanceErrors.ErrUnauthorized, w)
			return
		}

		tokenString := auth.ExtractToken(authHeader)
		claims, err := auth.VerifyToken(tokenString, secretKey)
		if err != nil {
			encodeError(r.Context(), err, w)
			return
		}

		// Store claims in context and proceed
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AllowMethods(next http.Handler, allowedMethods ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed := make(map[string]struct{})
		for _, method := range allowedMethods {
			allowed[method] = struct{}{}
		}
		if _, ok := allowed[r.Method]; !ok {
			encodeError(r.Context(), governanceErrors.ErrMethodNotAllowed, w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

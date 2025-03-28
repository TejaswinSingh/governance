package transport

import (
	"context"
	"net/http"

	"github.com/TejaswinSingh/version0/services/governance"
	"github.com/TejaswinSingh/version0/services/governance/auth"
	"github.com/TejaswinSingh/version0/services/governance/errors"
	myredis "github.com/TejaswinSingh/version0/services/governance/redis"
)

func JWTAuthMiddleware(secretKey string, redisService *myredis.RedisService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			encodeError(r.Context(), errors.ErrUnauthorized, w)
			return
		}

		// verify token
		tokenString := auth.ExtractToken(authHeader)
		claims, err := auth.VerifyToken(tokenString, secretKey)
		if err != nil {
			encodeError(r.Context(), err, w)
			return
		}

		// check that token is not blacklisted
		val, err := redisService.Get(r.Context(), tokenString)
		if err != nil && err != errors.ErrRedisKeyNotExist {
			encodeError(r.Context(), err, w)
			return
		}
		if err == nil && val == "blacklisted" {
			encodeError(r.Context(), errors.ErrInvalidToken, w)
			return
		}

		// Store claims and token in context and proceed
		ctx := context.WithValue(r.Context(), governance.ClaimsKey, claims)
		ctx = context.WithValue(ctx, governance.TokenKey, tokenString)
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
			encodeError(r.Context(), errors.ErrMethodNotAllowed, w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

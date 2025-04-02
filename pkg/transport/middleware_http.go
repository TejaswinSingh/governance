package transport

import (
	"context"
	"net/http"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/redis"

	"github.com/golang-jwt/jwt/v5"
)

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

func JWTAuthMiddleware(secretKey string, redisService *redis.RedisService, casbinService *auth.CasbinService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var claims jwt.MapClaims

		// authentication
		switch r.URL.Path {
		case "/auth/refresh":
			// get refresh-token
			token, err := getTokenFromCookie(r, "refresh_token")
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			tokenType := "refresh"
			claims, err = auth.VerifyToken(r.Context(), redisService, token, secretKey, tokenType)
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			r = setContext(r, governance.RefreshClaimsKey, claims, governance.RefreshTokenKey, token)

		case "/auth/logout":
			// get refresh-token
			token, err := getTokenFromCookie(r, "refresh_token")
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			tokenType := "refresh"
			claims, err = auth.VerifyToken(r.Context(), redisService, token, secretKey, tokenType)
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			r = setContext(r, governance.RefreshClaimsKey, claims, governance.RefreshTokenKey, token)
			// get access-token
			token, err = getTokenFromHeader(r)
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			tokenType = "access"
			claims, err = auth.VerifyToken(r.Context(), redisService, token, secretKey, tokenType)
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			r = setContext(r, governance.AccessClaimsKey, claims, governance.AccessTokenKey, token)

		default:
			// get access-token
			token, err := getTokenFromHeader(r)
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			tokenType := "access"
			claims, err = auth.VerifyToken(r.Context(), redisService, token, secretKey, tokenType)
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			r = setContext(r, governance.AccessClaimsKey, claims, governance.AccessTokenKey, token)
		}

		// authorization
		{
			role, _ := claims["aud"].(string)
			ok, err := casbinService.CheckPermission(role, r.URL.Path, r.Method)
			logger.Log("role", role, "path", r.URL.Path, "method", r.Method)
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			if !ok {
				encodeError(r.Context(), errors.ErrAccessDenied, w)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// helpers
func getTokenFromCookie(r *http.Request, cookieName string) (string, *errors.ServiceError) {
	cookie, err := r.Cookie(cookieName)
	if err != nil && cookieName == "refresh_token" {
		return "", errors.ErrNoRefreshCookie
	}
	return cookie.Value, nil
}

func getTokenFromHeader(r *http.Request) (string, *errors.ServiceError) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.ErrUnauthorized
	}
	return auth.ExtractToken(authHeader), nil
}

func setContext(r *http.Request, claimsKey governance.ContextKey, claimsValue jwt.MapClaims, tokenKey governance.ContextKey, tokenValue string) *http.Request {
	ctx := context.WithValue(r.Context(), claimsKey, claimsValue)
	ctx = context.WithValue(ctx, tokenKey, tokenValue)
	return r.WithContext(ctx)
}

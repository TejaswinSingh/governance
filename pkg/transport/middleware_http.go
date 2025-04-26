package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/persist"

	"github.com/golang-jwt/jwt/v5"
)

// CorsOptions defines the structure for CORS configuration options
type CorsOptions struct {
	Origin           string   // Origin specifies the origin that is allowed to access the resource
	Methods          []string // Methods is a list of HTTP methods allowed when accessing the resource
	Headers          []string // Headers is a list of HTTP headers that can be used when making the actual request
	ExposeHeaders    []string // ExposeHeaders indicates which headers are safe to expose to the API of a CORS API specification
	AllowCredentials bool     // AllowCredentials indicates whether or not the response to the request can be exposed when the credentials flag is true
}

func CORSMiddleware(corsConfig *CorsOptions, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsConfig.Origin)
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(corsConfig.Methods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsConfig.Headers, ", "))
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposeHeaders, ", "))
		w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprint(corsConfig.AllowCredentials))

		// preflight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func AllowMethods(next http.Handler, allowedMethods ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed := make(map[string]struct{})
		for _, method := range allowedMethods {
			allowed[method] = struct{}{}
		}
		if _, ok := allowed[r.Method]; !ok {
			encodeError(r.Context(), myerrors.ErrMethodNotAllowed, w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func JWTAuthMiddleware(secretKey string, redisService *persist.RedisService, casbinService *auth.CasbinService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			claims jwt.MapClaims
			err    error
		)

		// authentication
		switch r.URL.Path {
		case "/auth/refresh":
			// refresh token
			r, claims, err = extractAndVerifyToken(r, redisService, secretKey, "refresh")
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}

		case "/auth/logout":
			// access token
			r, claims, err = extractAndVerifyToken(r, redisService, secretKey, "access")
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
			// refresh token
			var claims2 jwt.MapClaims
			var err2 error
			r, claims2, err2 = extractAndVerifyToken(r, redisService, secretKey, "refresh")
			if err2 != nil {
				encodeError(r.Context(), err2, w)
				return
			}
			// check that both tokens have the same claims
			if !auth.CompareJWTClaims(claims, claims2, "sub", "iss", "aud") {
				encodeError(r.Context(), myerrors.ErrMismatchedClaims, w)
				return
			}

		default:
			// access token
			r, claims, err = extractAndVerifyToken(r, redisService, secretKey, "access")
			if err != nil {
				encodeError(r.Context(), err, w)
				return
			}
		}

		// authorization
		role, _ := claims["aud"].(string)
		ok, err := casbinService.CheckPermission(role, r.URL.Path, r.Method)
		logger.Log("role", role, "path", r.URL.Path, "method", r.Method)
		if err != nil { // casbin errors
			encodeError(r.Context(), err, w)
			return
		}
		if !ok { // denied
			encodeError(r.Context(), myerrors.ErrAccessDenied, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// helpers
func extractAndVerifyToken(r *http.Request, redisService *persist.RedisService, secretKey, tokenType string) (*http.Request, jwt.MapClaims, error) {
	if tokenType == "refresh" {
		token, err := getTokenFromHeader(r, "X-Refresh-Token")
		if err != nil {
			return r, nil, myerrors.ErrNoRefreshToken
		}
		claims, err2 := auth.VerifyToken(r.Context(), redisService, token, secretKey, tokenType)
		if err2 != nil {
			return r, nil, err2
		}
		// add tokens and claims to request.context
		// this makes them available inside request decoders
		return setContext(r, governance.RefreshClaimsKey, claims, governance.RefreshTokenKey, token), claims, nil
	} else {
		tokenType = "access"
		token, err := getTokenFromHeader(r, "Authorization")
		if err != nil {
			return r, nil, myerrors.ErrUnauthorized
		}
		claims, err2 := auth.VerifyToken(r.Context(), redisService, token, secretKey, tokenType)
		if err2 != nil {
			return r, nil, err2
		}
		return setContext(r, governance.AccessClaimsKey, claims, governance.AccessTokenKey, token), claims, nil
	}
}

func getTokenFromHeader(r *http.Request, headerKey string) (string, error) {
	headerVal := r.Header.Get(headerKey)
	if headerVal == "" {
		return "", errors.New("missing header key")
	}
	if headerKey == "Authorization" {
		return auth.ExtractToken(headerVal), nil
	}
	return headerVal, nil
}

func setContext(r *http.Request, claimsKey governance.ContextKey, claimsValue jwt.MapClaims, tokenKey governance.ContextKey, tokenValue string) *http.Request {
	ctx := context.WithValue(r.Context(), claimsKey, claimsValue)
	ctx = context.WithValue(ctx, tokenKey, tokenValue)
	return r.WithContext(ctx)
}

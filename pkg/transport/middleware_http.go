package transport

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/config"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/Resolyte-official/service-governance/pkg/metrics"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
)

// CorsOptions defines the structure for CORS configuration options
type CorsOptions struct {
	Origins          []string // Origins is a list of origins that are allowed to access the resource
	Methods          []string // Methods is a list of HTTP methods allowed when accessing the resource
	Headers          []string // Headers is a list of HTTP headers that can be used when making the actual request
	ExposeHeaders    []string // ExposeHeaders indicates which headers are safe to expose to the API of a CORS API specification
	AllowCredentials bool     // AllowCredentials indicates whether or not the response to the request can be exposed when the credentials flag is true
}

func CORS(corsConfig *CorsOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			origin := r.Header.Get("Origin")

			if origin == "" || !slices.Contains(corsConfig.Origins, origin) {
				if r.Method == http.MethodOptions { // preflight request
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// only set CORS Headers for allowed origins
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(corsConfig.Methods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsConfig.Headers, ", "))
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposeHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprint(corsConfig.AllowCredentials))

			// preflight request
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func TracingLoggingMetricsMiddleware(logger logging.Logger, envConfig *config.Configuration, metrics *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now().UTC()

			rw := &ResponseWriter{ResponseWriter: w, status: http.StatusOK}

			logE := &LogEntry{
				Transport: string(envConfig.HttpMode),
				Method:    r.Method,
				Path:      r.URL.Path,
			}
			defer func() {
				logE.Status = rw.status
				logE.Error = rw.Err
				logE.Duration = time.Since(start).String()
				logE.Log(logger)

				// metrics
				labels := []string{
					"transport", logE.Transport,
					"path", logE.Path,
					"rpc", "",
					"status", strconv.Itoa(logE.Status),
				}
				metrics.RequestLatency.With(labels...).Observe(time.Since(start).Seconds())
				metrics.RequestCount.With(labels...).Add(1)
			}()

			// don't trace preflight request and request to metrics endpoint
			// TODO: this check can be removed in prod
			if r.Method != http.MethodOptions && r.URL.Path != "/metrics" {

				// requestID is required
				reqID := r.Header.Get("X-Request-ID")
				if reqID == "" {
					encodeError(r.Context(), myerrors.ErrMissingRequestID, rw)
					return
				}
				if uuid.Validate(reqID) != nil {
					encodeError(r.Context(), myerrors.ErrInvalidRequestID, rw)
					return
				}
				logE.RequestID = reqID

				// try to obtain userID from token, wherever possible
				switch r.URL.Path {
				case "/auth/refresh":
					if token, err := auth.ReadEncryptedCookie(r, string(auth.RefreshToken), envConfig.SecretKey); err == nil {
						if claims, err := auth.ExtractClaims(token); err == nil {
							if sub, ok := claims["sub"].(string); ok {
								logE.UserID = sub
							}
						}
					}
				case "/governance/healthz", "/auth/request-otp", "/auth/login", "/metrics", "/auth/school-admin/token":
					// no token in header
				default:
					if token, err := auth.ReadEncryptedCookie(r, string(auth.AccessToken), envConfig.SecretKey); err == nil {
						if claims, err := auth.ExtractClaims(token); err == nil {
							if sub, ok := claims["sub"].(string); ok {
								logE.UserID = sub
							}
						}
					}
				}
			}

			// call next handler
			next.ServeHTTP(rw, r)
		})
	}
}

func AuthMiddleware(envConfig *config.Configuration, svcStore *governance.ServiceStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				claims jwt.MapClaims
				err    error
			)

			// authentication
			switch r.URL.Path {

			// unprotected paths
			case "/governance/healthz", "/auth/request-otp", "/auth/login", "/auth/school-admin/token", "/metrics":
				next.ServeHTTP(w, r)
				return

			case "/auth/refresh":
				// refresh token
				r, claims, err = extractAndVerifyToken(r, svcStore.Redis, envConfig.SecretKey, auth.RefreshToken)
				if err != nil {
					encodeError(r.Context(), err, w)
					return
				}

			case "/auth/logout":
				// access token
				r, claims, err = extractAndVerifyToken(r, svcStore.Redis, envConfig.SecretKey, auth.AccessToken)
				if err != nil {
					encodeError(r.Context(), err, w)
					return
				}
				// refresh token
				var claims2 jwt.MapClaims
				var err2 error
				r, claims2, err2 = extractAndVerifyToken(r, svcStore.Redis, envConfig.SecretKey, auth.RefreshToken)
				if err2 != nil {
					encodeError(r.Context(), err2, w)
					return
				}
				// check that both tokens have the same claims
				if !auth.CompareJWTClaims(claims, claims2, "sub", "iss", "aud", "role") {
					encodeError(r.Context(), myerrors.ErrMismatchedClaims, w)
					return
				}

			case "/auth/school-admin/metadata":
				// access token
				r, claims, err = extractAndVerifyToken(r, svcStore.Redis, envConfig.SecretKey, auth.AccessToken)
				if err != nil {
					encodeError(r.Context(), err, w)
					return
				}
			default:
				// 404 paths
				next.ServeHTTP(w, r)
				return
			}

			// authorization
			role, _ := claims["role"].(string)
			ok, err := svcStore.Casbin.CheckPermission(role, r.URL.Path, r.Method)
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
}

// helpers
func extractAndVerifyToken(r *http.Request, redisService *persist.RedisService, secretKey []byte, tk auth.AuthToken) (*http.Request, jwt.MapClaims, error) {
	switch tk {
	case auth.RefreshToken:
		token, err := auth.ReadEncryptedCookie(r, string(auth.RefreshToken), secretKey)
		if err != nil {
			return r, nil, err
		}
		claims, err2 := auth.VerifyToken(r.Context(), redisService, token, secretKey, tk)
		if err2 != nil {
			return r, nil, err2
		}
		// add tokens and claims to request.context
		// this makes them available inside request decoders
		return setContext(r, RefreshClaimsKey, claims, RefreshTokenKey, token), claims, nil
	default: // auth.AccessToken
		token, err := auth.ReadEncryptedCookie(r, string(auth.AccessToken), secretKey)
		if err != nil {
			return r, nil, err
		}
		claims, err2 := auth.VerifyToken(r.Context(), redisService, token, secretKey, tk)
		if err2 != nil {
			return r, nil, err2
		}
		return setContext(r, AccessClaimsKey, claims, AccessTokenKey, token), claims, nil
	}
}

func setContext(r *http.Request, claimsKey ContextKey, claimsValue jwt.MapClaims, tokenKey ContextKey, tokenValue string) *http.Request {
	ctx := context.WithValue(r.Context(), claimsKey, claimsValue)
	ctx = context.WithValue(ctx, tokenKey, tokenValue)
	return r.WithContext(ctx)
}

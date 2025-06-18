package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/config"
	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	governanceErrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/Resolyte-official/service-governance/pkg/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ContextKey string

const (
	AccessClaimsKey  ContextKey = "access-claims"
	RefreshClaimsKey ContextKey = "refresh-claims"
	AccessTokenKey   ContextKey = "access-token"
	RefreshTokenKey  ContextKey = "refresh-token"
)

func NewHTTPHandler(ep endpoints.Set, envConfig *config.Configuration, corsConfig *CorsOptions, svcStore *governance.ServiceStore, metrics *metrics.Metrics, lw logging.LogWriter) http.Handler {
	m := http.NewServeMux()

	m.Handle("/metrics", promhttp.Handler())

	m.Handle("GET /governance/healthz", httptransport.NewServer(
		ep.ServiceStatusEndpoint,
		decodeHTTPServiceStatusRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError), // called automatically if error occurs in either of the following: decoder, endpoint, encoder
	))

	m.Handle("POST /auth/login", httptransport.NewServer(
		ep.LoginEndpoint,
		decodeHTTPLoginRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	))

	m.Handle("POST /auth/logout", httptransport.NewServer(
		ep.LogoutEndpoint,
		decodeHTTPLogoutRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	))

	m.Handle("POST /auth/refresh", httptransport.NewServer(
		ep.RefreshEndpoint,
		decodeHTTPRefreshRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	))

	logger := logging.NewContextLogger(lw, "HttpHandler")

	handler := AuthMiddleware(envConfig, svcStore)(m)
	handler = CORS(corsConfig)(handler)
	handler = TracingLoggingMetricsMiddleware(logger, envConfig, metrics)(handler)

	return handler
}

func decodeHTTPServiceStatusRequest(_ context.Context, _ *http.Request) (any, error) {
	return endpoints.ServiceStatusRequest{}, nil
}

func decodeHTTPLoginRequest(_ context.Context, r *http.Request) (any, error) {
	var req endpoints.LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		return nil, governanceErrors.ErrBadRequest
	}
	return req, nil
}

func decodeHTTPLogoutRequest(ctx context.Context, r *http.Request) (any, error) {
	return endpoints.LogoutRequest{
		// get values extracted by the auth middleware
		TokenAccess:   ctx.Value(AccessTokenKey).(string),
		TokenRefresh:  ctx.Value(RefreshTokenKey).(string),
		ClaimsAccess:  ctx.Value(AccessClaimsKey).(jwt.MapClaims),
		ClaimsRefresh: ctx.Value(RefreshClaimsKey).(jwt.MapClaims),
	}, nil
}

func decodeHTTPRefreshRequest(ctx context.Context, r *http.Request) (any, error) {
	return endpoints.RefreshRequest{
		TokenRefresh:  ctx.Value(RefreshTokenKey).(string),
		ClaimsRefresh: ctx.Value(RefreshClaimsKey).(jwt.MapClaims),
	}, nil
}

func makeEncodeResponse(envConfig *config.Configuration) httptransport.EncodeResponseFunc {
	return func(ctx context.Context, w http.ResponseWriter, response any) error {
		if e, ok := response.(error); ok && e != nil {
			return e
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// set auth cookies for login and refresh response
		if resp, ok := response.(interface {
			GetAccessToken() string
			GetRefreshToken() string
		}); ok {
			auth.SetAuthCookie(w, envConfig.SecretKey, auth.AccessToken, resp.GetAccessToken())
			auth.SetAuthCookie(w, envConfig.SecretKey, auth.RefreshToken, resp.GetRefreshToken())
		}

		// clear auth cookies for LogoutResponse
		if _, ok := response.(endpoints.LogoutResponse); ok {
			auth.ClearAuthCookie(w, auth.AccessToken)
			auth.ClearAuthCookie(w, auth.RefreshToken)
		}

		// write status in header
		status := http.StatusOK
		if resp, ok := response.(interface{ GetStatus() int }); ok {
			status = resp.GetStatus()
		}
		w.WriteHeader(status)

		if status == http.StatusNoContent {
			return nil
		}
		return json.NewEncoder(w).Encode(response)
	}
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw, _ := w.(*ResponseWriter) // custom response writer

	var serviceErr *governanceErrors.ServiceError
	if errors.As(err, &serviceErr) {
		rw.Err = serviceErr.DetailedError()
		w.WriteHeader(serviceErr.HTTPCode)
		json.NewEncoder(w).Encode(serviceErr)
		return
	}

	// default to internal server error if it is unknown
	// but still log the original error
	defErr := governanceErrors.ErrInternalServer
	rw.Err = fmt.Sprintf("%s [Internal Error: %s]", defErr.Error(), err.Error())
	w.WriteHeader(defErr.HTTPCode)
	json.NewEncoder(w).Encode(defErr)
}

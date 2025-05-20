package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	governanceErrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/golang-jwt/jwt/v5"
)

func NewHTTPHandler(ep endpoints.Set, secretKey string, corsConfig *CorsOptions, redisService *persist.RedisService, casbinService *auth.CasbinService, lw logging.LogWriter) http.Handler {
	logger := logging.NewContextLogger(lw, "HttpHandler")
	m := http.NewServeMux()

	m.Handle("/governance/healthz", LoggingMiddleware(logger)(TracingMiddleware(CORSMiddleware(corsConfig, AllowMethods(AuthMiddleware(secretKey, redisService, casbinService, httptransport.NewServer(
		ep.ServiceStatusEndpoint,
		decodeHTTPServiceStatusRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError), // this will be called if error occurs at any point
	)), "GET")))))

	m.Handle("/auth/login", LoggingMiddleware(logger)(TracingMiddleware(CORSMiddleware(corsConfig, AllowMethods(httptransport.NewServer(
		ep.LoginEndpoint,
		decodeHTTPLoginRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError),
	), "POST")))))

	m.Handle("/auth/logout", LoggingMiddleware(logger)(TracingMiddleware(CORSMiddleware(corsConfig, AllowMethods(AuthMiddleware(secretKey, redisService, casbinService, httptransport.NewServer(
		ep.LogoutEndpoint,
		decodeHTTPLogoutRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError),
	)), "POST")))))

	m.Handle("/auth/refresh", LoggingMiddleware(logger)(TracingMiddleware(CORSMiddleware(corsConfig, AllowMethods(AuthMiddleware(secretKey, redisService, casbinService, httptransport.NewServer(
		ep.RefreshEndpoint,
		decodeHTTPRefreshRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError),
	)), "POST")))))

	return m
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
		TokenAccess:   ctx.Value(governance.AccessTokenKey).(string),
		TokenRefresh:  ctx.Value(governance.RefreshTokenKey).(string),
		ClaimsAccess:  ctx.Value(governance.AccessClaimsKey).(jwt.MapClaims),
		ClaimsRefresh: ctx.Value(governance.RefreshClaimsKey).(jwt.MapClaims),
	}, nil
}

func decodeHTTPRefreshRequest(ctx context.Context, r *http.Request) (any, error) {
	return endpoints.RefreshRequest{
		TokenRefresh:  ctx.Value(governance.RefreshTokenKey).(string),
		ClaimsRefresh: ctx.Value(governance.RefreshClaimsKey).(jwt.MapClaims),
	}, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	if e, ok := response.(error); ok && e != nil {
		return e
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// set access token and refresh token in the header
	// for login and refresh response
	if resp, ok := response.(interface {
		GetAccessToken() string
		GetRefreshToken() string
	}); ok {
		w.Header().Set("Authorization", "Bearer "+resp.GetAccessToken())
		w.Header().Set("X-Refresh-Token", resp.GetRefreshToken())
	}

	// write status in header
	var status int
	if resp, ok := response.(interface{ GetStatus() int }); ok {
		status = resp.GetStatus()
	} else {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	if status == http.StatusNoContent {
		return nil
	}
	return json.NewEncoder(w).Encode(response)
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

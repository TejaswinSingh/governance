package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

	logger := logging.NewContextLogger(lw, "HttpHandler")

	wrap := func(h http.Handler) http.Handler {
		return TracingLoggingMetricsMiddleware(logger, envConfig, metrics)(
			AuthMiddleware(envConfig, svcStore)(h),
		)
	}

	m.Handle("/metrics", wrap(promhttp.Handler()))

	m.Handle("GET /healthz", wrap(httptransport.NewServer(
		ep.ServiceStatusEndpoint,
		decodeHTTPServiceStatusRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError), // called automatically if error occurs in either of the following: decoder, endpoint, encoder
	)))

	m.Handle("POST /school-admin/token", wrap(httptransport.NewServer(
		ep.GetSchoolAdminTokenEndpoint,
		decodeGetSchoolAdminTokenRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	m.Handle("POST /school-admin/metadata", wrap(httptransport.NewServer(
		ep.GetSchoolAdminMetadataEndpoint,
		decodeGetSchoolAdminMetadataRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	m.Handle("POST /request-otp", wrap(httptransport.NewServer(
		ep.SendOTPEndpoint,
		decodeHTTPSendOTPRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	m.Handle("POST /login", wrap(httptransport.NewServer(
		ep.LoginEndpoint,
		decodeHTTPLoginRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	m.Handle("POST /logout", wrap(httptransport.NewServer(
		ep.LogoutEndpoint,
		decodeHTTPLogoutRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	m.Handle("POST /refresh", wrap(httptransport.NewServer(
		ep.RefreshEndpoint,
		decodeHTTPRefreshRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	m.Handle("POST /access/delegate", wrap(httptransport.NewServer(
		ep.DelegateAccessEndpoint,
		decodeHTTPDelegateAccessRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	m.Handle("POST /access/revoke", wrap(httptransport.NewServer(
		ep.RevokeAccessEndpoint,
		decodeHTTPRevokeAccessRequest,
		makeEncodeResponse(envConfig),
		httptransport.ServerErrorEncoder(encodeError),
	)))

	// match all: return 404
	m.Handle("/", wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "404 not found", http.StatusNotFound)
	})))

	return CORS(corsConfig)(m)
}

func decodeHTTPServiceStatusRequest(_ context.Context, _ *http.Request) (any, error) {
	return endpoints.ServiceStatusRequest{}, nil
}

func decodeGetSchoolAdminTokenRequest(ctx context.Context, r *http.Request) (any, error) {
	var req endpoints.GetSchoolAdminTokenRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		return nil, governanceErrors.ErrBadRequest
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		return nil, governanceErrors.ErrBadRequest
	}
	return req, nil
}

func decodeGetSchoolAdminMetadataRequest(ctx context.Context, r *http.Request) (any, error) {
	return endpoints.GetSchoolAdminMetadataRequest{
		// get values extracted by the auth middleware
		Claims: ctx.Value(AccessClaimsKey).(jwt.MapClaims),
	}, nil
}

func decodeHTTPSendOTPRequest(_ context.Context, r *http.Request) (any, error) {
	var req endpoints.SendOTPRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		return nil, governanceErrors.ErrBadRequest
	}
	req.CountryCode = strings.TrimSpace(req.CountryCode)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.CountryCode == "" || req.Phone == "" {
		return nil, governanceErrors.ErrBadRequest
	}
	return req, nil
}

func decodeHTTPLoginRequest(_ context.Context, r *http.Request) (any, error) {
	var req endpoints.LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		return nil, governanceErrors.ErrBadRequest
	}
	req.CountryCode = strings.TrimSpace(req.CountryCode)
	req.Phone = strings.TrimSpace(req.Phone)
	req.OTP = strings.TrimSpace(req.OTP)
	if req.CountryCode == "" || req.Phone == "" || req.OTP == "" {
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

func decodeHTTPDelegateAccessRequest(ctx context.Context, r *http.Request) (any, error) {
	var req endpoints.DelegateAccessRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		return nil, governanceErrors.ErrBadRequest
	}
	req.GuardianPhone = strings.TrimSpace(req.GuardianPhone)
	if req.GuardianPhone == "" || len(req.StudentIDs) < 1 {
		return nil, governanceErrors.ErrBadRequest
	}
	claims, ok := ctx.Value(AccessClaimsKey).(jwt.MapClaims)
	if !ok {
		return nil, governanceErrors.ErrBadRequest
	}
	req.ParentID, ok = claims["sub"].(string)
	if !ok {
		return nil, governanceErrors.ErrBadRequest
	}
	return req, nil
}

func decodeHTTPRevokeAccessRequest(ctx context.Context, r *http.Request) (any, error) {
	var req endpoints.RevokeAccessRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		return nil, governanceErrors.ErrBadRequest
	}
	req.GuardianID = strings.TrimSpace(req.GuardianID)
	if req.GuardianID == "" || len(req.StudentIDs) < 1 {
		return nil, governanceErrors.ErrBadRequest
	}
	claims, ok := ctx.Value(AccessClaimsKey).(jwt.MapClaims)
	if !ok {
		return nil, governanceErrors.ErrBadRequest
	}
	req.ParentID, ok = claims["sub"].(string)
	if !ok {
		return nil, governanceErrors.ErrBadRequest
	}
	return req, nil
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

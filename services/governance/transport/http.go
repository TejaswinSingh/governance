package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/TejaswinSingh/version0/services/governance/endpoints"
	governanceErrors "github.com/TejaswinSingh/version0/services/governance/errors"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

func NewHTTPHandler(ep endpoints.Set, secretKey string) http.Handler {
	m := http.NewServeMux()

	m.Handle("/governance/healthz", AllowMethods(JWTAuthMiddleware(secretKey, httptransport.NewServer(
		ep.ServiceStatusEndpoint,
		decodeHTTPServiceStatusRequest,
		encodeResponse,
	)), "GET"))

	m.Handle("/login", AllowMethods(httptransport.NewServer(
		ep.LoginEndpoint,
		decodeHTTPLoginRequest,
		encodeResponse,
		httptransport.ServerErrorEncoder(encodeError), // errors occured while decoding will be passed to this
	), "POST"))

	return m
}

func decodeHTTPServiceStatusRequest(_ context.Context, _ *http.Request) (any, error) {
	var req endpoints.ServiceStatusRequest
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
	return req, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	if e, ok := response.(error); ok && e != nil {
		encodeError(ctx, e, w)
		return nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// set JWT in header for LoginResponse
	if loginResponse, ok := response.(endpoints.LoginResponse); ok {
		w.Header().Set("Authorization", "Bearer "+loginResponse.Token)
		w.WriteHeader(http.StatusOK) // header must be set before a call to WriteHeader
		logger.Log("Response", response)
		return nil
	}

	w.WriteHeader(http.StatusOK)
	logger.Log("Response", response) // log the response
	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var serviceErr *governanceErrors.ServiceError
	if errors.As(err, &serviceErr) {
		w.WriteHeader(serviceErr.HTTPCode)
		json.NewEncoder(w).Encode(serviceErr)
		logger.Log("Error", serviceErr.Error()) // log the error
		return
	}

	// default to internal server error if error is unknown
	error := governanceErrors.ErrInternalServer
	w.WriteHeader(error.HTTPCode)
	json.NewEncoder(w).Encode(error)
	logger.Log("Error", error.Error())
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}

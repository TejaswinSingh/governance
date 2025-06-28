package errors

import (
	"fmt"
	"net/http"
)

type ServiceError struct {
	InternalErr error  `json:"-"`
	HTTPCode    int    `json:"-"`
	ErrorCode   string `json:"errorCode"`
	Message     string `json:"message"`
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
}

func (e *ServiceError) WithInternalErr(err error) *ServiceError {
	return &ServiceError{
		InternalErr: err,
		HTTPCode:    e.HTTPCode,
		ErrorCode:   e.ErrorCode,
		Message:     e.Message,
	}
}

func (e *ServiceError) Is(target error) bool {
	t, ok := target.(*ServiceError)
	if !ok {
		return false
	}
	return e.ErrorCode == t.ErrorCode
}

func (e *ServiceError) Unwrap() error {
	return e.InternalErr
}

func (e *ServiceError) DetailedError() string {
	if e.InternalErr != nil {
		return fmt.Sprintf("%s: %s [Internal Error: %s]", e.ErrorCode, e.Message, e.InternalErr.Error())
	} else {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
	}
}

func NewServiceError(httpCode int, errorCode, message string) *ServiceError {
	return &ServiceError{HTTPCode: httpCode, ErrorCode: errorCode, Message: message}
}

var (
	// standard errors
	ErrBadRequest         = NewServiceError(http.StatusBadRequest, "BAD_REQUEST", "The request could not be processed due to malformed syntax or missing required parameters.")
	ErrUnauthorized       = NewServiceError(http.StatusUnauthorized, "UNAUTHORIZED_REQUEST", "Unauthorized request. Please provide a valid Authorization token/cookie.")
	ErrInternalServer     = NewServiceError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrNotFound           = NewServiceError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "The requested resource was not found.")
	ErrServiceUnavailable = NewServiceError(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The service is temporarily unavailable. Please try again later.")

	// jwt errors
	ErrCreateJWT = NewServiceError(http.StatusInternalServerError, "JWT_CREATION_ERROR", "Failed to generate a token due to an internal processing error. Please try again or contact support if the issue persists.")

	// redis errors
	ErrRedis            = NewServiceError(http.StatusInternalServerError, "REDIS_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrRedisStopped     = NewServiceError(http.StatusInternalServerError, "REDIS_STOPPED", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrRedisKeyNotExist = NewServiceError(http.StatusNotFound, "REDIS_KEY_NOT_FOUND", "The requested key does not exist in Redis.")

	// mongo errors
	ErrMongo = NewServiceError(http.StatusInternalServerError, "MONGO_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")

	// meta data errors
	ErrMissingMetaData = NewServiceError(http.StatusNotFound, "MISSING_METADATA", "Required metadata was not found on the server.")
	ErrMetaData        = NewServiceError(http.StatusInternalServerError, "METADATA_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")

	// grpc errors
	ErrGRPCBadRequest = NewServiceError(http.StatusBadRequest, "GRPC_BAD_REQUEST", "Missing required fields in gRPC request.")

	// auth-N errors
	ErrInvalidCreds      = NewServiceError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "The provided username or password is incorrect. Please check your credentials and try again.")
	ErrInvalidSignMethod = NewServiceError(http.StatusUnauthorized, "INVALID_SIGN_METHOD", "The token's signing method is invalid or does not match the expected algorithm. Ensure that the correct signing algorithm is used.")
	ErrInvalidToken      = NewServiceError(http.StatusUnauthorized, "INVALID_TOKEN", "The provided token is invalid, malformed, or has been tampered with. Please provide a valid token and try again.")
	ErrInvalidClaims     = NewServiceError(http.StatusUnauthorized, "INVALID_CLAIMS", "The token's claims are invalid, missing required fields, or have an incorrect format. Ensure that the token contains valid claims.")
	ErrMismatchedClaims  = NewServiceError(http.StatusUnauthorized, "MISMATCHED_CLAIMS", "The claims in the access and refresh tokens do not match. Please re-authenticate.")
	ErrInvalidOTP        = NewServiceError(http.StatusUnauthorized, "INVALID_OTP", "Unauthorized request. Please provide a valid OTP.")

	// auth-Z errors
	ErrAccessDenied      = NewServiceError(http.StatusForbidden, "ACCESS_DENIED", "You do not have permission to perform this action.")
	ErrCasbinPolicyLoad  = NewServiceError(http.StatusInternalServerError, "CASBIN_POLICY_LOAD_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrCasbinEnforcement = NewServiceError(http.StatusInternalServerError, "CASBIN_ENFORCEMENT_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")

	// others
	ErrMissingRequestID = NewServiceError(http.StatusBadRequest, "MISSING_REQUEST_ID", "Please provide a valid X-Request-ID header.")
	ErrInvalidRequestID = NewServiceError(http.StatusBadRequest, "INVALID_REQUEST_ID", "X-Request-ID must be a valid UUID.")
)

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

func (e *ServiceError) SetInternalErr(err error) *ServiceError {
	e.InternalErr = err
	return e
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
	ErrBadRequest       = NewServiceError(http.StatusBadRequest, "BAD_REQUEST", "The request could not be processed due to malformed syntax or missing required parameters.")
	ErrUnauthorized     = NewServiceError(http.StatusUnauthorized, "UNAUTHORIZED_REQUEST", "Unauthorized request. Please provide a valid Authorization header.")
	ErrMethodNotAllowed = NewServiceError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The requested HTTP method is not allowed for this endpoint.")
	ErrInternalServer   = NewServiceError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")

	// jwt errors
	ErrCreateJWT = NewServiceError(http.StatusInternalServerError, "JWT_CREATION_ERROR", "Failed to generate a token due to an internal processing error. Please try again or contact support if the issue persists.")

	// redis errors
	ErrRedis            = NewServiceError(http.StatusInternalServerError, "REDIS_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrRedisStopped     = NewServiceError(http.StatusInternalServerError, "REDIS_STOPPED", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrRedisKeyNotExist = NewServiceError(http.StatusNotFound, "REDIS_KEY_NOT_FOUND", "The requested key does not exist in Redis.")

	// mongo errors
	ErrMongo = NewServiceError(http.StatusInternalServerError, "MONGO_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")

	// meta data errors
	ErrMetaData = NewServiceError(http.StatusInternalServerError, "METADATA_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")

	// grpc errors
	ErrGRPCBadRequest = NewServiceError(http.StatusBadRequest, "GRPC_BAD_REQUEST", "Missing required fields in gRPC request.")

	// auth-N errors
	ErrInvalidCreds      = NewServiceError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "The provided username or password is incorrect. Please check your credentials and try again.")
	ErrMissingCreds      = NewServiceError(http.StatusUnauthorized, "MISSING_CREDENTIALS", "Username and password are required fields. Please provide both and try again.")
	ErrInvalidSignMethod = NewServiceError(http.StatusUnauthorized, "INVALID_SIGN_METHOD", "The token's signing method is invalid or does not match the expected algorithm. Ensure that the correct signing algorithm is used.")
	ErrInvalidToken      = NewServiceError(http.StatusUnauthorized, "INVALID_TOKEN", "The provided token is invalid, malformed, or has been tampered with. Please provide a valid token and try again.")
	ErrInvalidClaims     = NewServiceError(http.StatusUnauthorized, "INVALID_CLAIMS", "The token's claims are invalid, missing required fields, or have an incorrect format. Ensure that the token contains valid claims.")
	ErrNoRefreshToken    = NewServiceError(http.StatusUnauthorized, "NO_REFRESH_TOKEN", "Unauthorized request. Please provide a valid X-Refresh-Token header.")
	ErrMismatchedClaims  = NewServiceError(http.StatusUnauthorized, "MISMATCHED_CLAIMS", "The claims in the access and refresh tokens do not match. Please re-authenticate.")

	// auth-Z errors
	ErrAccessDenied      = NewServiceError(http.StatusForbidden, "ACCESS_DENIED", "You do not have permission to perform this action.")
	ErrCasbinPolicyLoad  = NewServiceError(http.StatusInternalServerError, "CASBIN_POLICY_LOAD_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrCasbinEnforcement = NewServiceError(http.StatusInternalServerError, "CASBIN_ENFORCEMENT_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
)

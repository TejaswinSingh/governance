package errors

import (
	"fmt"
	"net/http"
)

type ServiceError struct {
	HTTPCode  int    `json:"-"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
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
	ErrCreateJWT = NewServiceError(http.StatusInternalServerError, "JWT_CREATION_ERROR", "Failed to generate a token due to an internal processing error. Please try again or contact support.")

	// redis errors
	ErrRedis            = NewServiceError(http.StatusInternalServerError, "REDIS_ERROR", "A Redis-related error occurred. Please try again later.")
	ErrRedisStopped     = NewServiceError(http.StatusInternalServerError, "REDIS_STOPPED", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrRedisKeyNotExist = NewServiceError(http.StatusNotFound, "REDIS_KEY_NOT_FOUND", "The requested key does not exist in Redis.")

	// auth	errors
	ErrInvalidCreds      = NewServiceError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "The provided username or password is incorrect. Please check your credentials and try again.")
	ErrMissingCreds      = NewServiceError(http.StatusUnauthorized, "MISSING_CREDENTIALS", "Username and password are required fields. Please provide both and try again.")
	ErrInvalidSignMethod = NewServiceError(http.StatusUnauthorized, "INVALID_SIGN_METHOD", "The token's signing method is invalid or does not match the expected algorithm. Ensure that the correct signing algorithm is used.")
	ErrInvalidToken      = NewServiceError(http.StatusUnauthorized, "INVALID_TOKEN", "The provided token is invalid, malformed, or has been tampered with. Please provide a valid token and try again.")
	ErrInvalidClaims     = NewServiceError(http.StatusUnauthorized, "INVALID_CLAIMS", "The token's claims are invalid, missing required fields, or have an incorrect format. Ensure that the token contains valid claims.")
	ErrNoRefreshCookie   = NewServiceError(http.StatusUnauthorized, "NO_REFRESH_COOKIE", "The refresh token cookie is missing. Please ensure that the refresh token is set in the cookie and try again.")
)

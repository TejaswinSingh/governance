package endpoints

import (
	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

type ServiceStatusRequest struct{}
type ServiceStatusResponse struct {
	Code int                  `json:"status"`
	Err  *errors.ServiceError `json:"err,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginResponse struct {
	TokenAccess  string               `json:"-"`
	TokenRefresh string               `json:"-"`
	Err          *errors.ServiceError `json:"err,omitempty"`
}

func (l LoginResponse) GetAccessToken() string  { return l.TokenAccess }
func (l LoginResponse) GetRefreshToken() string { return l.TokenRefresh }

type LogoutRequest struct {
	TokenAccess   string
	TokenRefresh  string
	ClaimsAccess  jwt.MapClaims
	ClaimsRefresh jwt.MapClaims
}
type LogoutResponse struct {
	Err *errors.ServiceError `json:"err,omitempty"`
}

type RefreshRequest struct {
	TokenRefresh  string
	ClaimsRefresh jwt.MapClaims
}
type RefreshResponse struct {
	TokenAccess  string               `json:"-"`
	TokenRefresh string               `json:"-"`
	Err          *errors.ServiceError `json:"err,omitempty"`
}

func (r RefreshResponse) GetAccessToken() string  { return r.TokenAccess }
func (r RefreshResponse) GetRefreshToken() string { return r.TokenRefresh }

package endpoints

import (
	"github.com/TejaswinSingh/version0/services/governance/errors"
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

type LogoutRequest struct{}

type LogoutResponse struct {
	Err *errors.ServiceError `json:"err,omitempty"`
}

type RefreshRequest struct{}

type RefreshResponse struct {
	TokenAccess  string               `json:"-"`
	TokenRefresh string               `json:"-"`
	Err          *errors.ServiceError `json:"err,omitempty"`
}

func (r RefreshResponse) GetAccessToken() string  { return r.TokenAccess }
func (r RefreshResponse) GetRefreshToken() string { return r.TokenRefresh }

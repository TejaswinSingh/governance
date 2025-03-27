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
	Token string               `json:"-"`
	Err   *errors.ServiceError `json:"err,omitempty"`
}

type LogoutRequest struct{}

type LogoutResponse struct {
	Err *errors.ServiceError `json:"err,omitempty"`
}

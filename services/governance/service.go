package governance

import (
	"context"

	"github.com/TejaswinSingh/version0/services/governance/errors"
)

type contextKey string

const (
	ClaimsKey contextKey = "claims"
	TokenKey  contextKey = "token"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, *errors.ServiceError)
	Login(ctx context.Context, username, password string) (string, *errors.ServiceError)
	Logout(ctx context.Context) *errors.ServiceError
}

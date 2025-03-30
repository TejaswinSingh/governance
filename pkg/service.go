package governance

import (
	"context"

	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

type ContextKey string

const (
	AccessClaimsKey  ContextKey = "access-claims"
	RefreshClaimsKey ContextKey = "refresh-claims"
	AccessTokenKey   ContextKey = "access-token"
	RefreshTokenKey  ContextKey = "refresh-token"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, *errors.ServiceError)
	Login(ctx context.Context, username, password string) (tokenAccess, tokenRefresh string, err *errors.ServiceError)
	Logout(ctx context.Context, tokenAccess, tokenRefresh string, claimsAccess, claimsRefresh jwt.MapClaims) *errors.ServiceError
	Refresh(ctx context.Context, curTokenRefresh string, claims jwt.MapClaims) (tokenAccess, tokenRefresh string, err *errors.ServiceError)
}

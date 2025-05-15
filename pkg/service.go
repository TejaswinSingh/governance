package governance

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ContextKey string

const (
	AccessClaimsKey  ContextKey = "access-claims"
	RefreshClaimsKey ContextKey = "refresh-claims"
	AccessTokenKey   ContextKey = "access-token"
	RefreshTokenKey  ContextKey = "refresh-token"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, error)
	Login(ctx context.Context, username, password string) (tokenAccess, tokenRefresh string, mdata *bson.M, status int, err error)
	Logout(ctx context.Context, tokenAccess, tokenRefresh string, claimsAccess, claimsRefresh jwt.MapClaims) (int, error)
	Refresh(ctx context.Context, curTokenRefresh string, claims jwt.MapClaims) (tokenAccess, tokenRefresh string, mdata *bson.M, status int, err error)
	Authenticate(ctx context.Context, token string) (int, error)
	Authorize(ctx context.Context, token, obj, act string) (bool, int, error)
}

package governance

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Service interface {
	ServiceStatus(ctx context.Context) (int, error)
	GetSchoolAdminToken(ctx context.Context, email string) (token string, status int, err error)
	GetSchoolAdminMetadata(ctx context.Context, claims jwt.MapClaims) (mdata *bson.M, status int, err error)
	SendOTP(ctx context.Context, countryCode, phone string) (int, error)
	Login(ctx context.Context, countryCode, phone, otp string) (tokenAccess, tokenRefresh string, mdata *bson.M, status int, err error)
	Logout(ctx context.Context, tokenAccess, tokenRefresh string, claimsAccess, claimsRefresh jwt.MapClaims) (int, error)
	Refresh(ctx context.Context, curTokenRefresh string, claims jwt.MapClaims) (tokenAccess, tokenRefresh string, mdata *bson.M, status int, err error)
	Authenticate(ctx context.Context, token string) (claims jwt.MapClaims, status int, err error)
	Authorize(ctx context.Context, token, obj, act string) (eft bool, claims jwt.MapClaims, status int, err error)
}

package governance

import (
	"context"
	"net/http"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/config"
	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type governanceService struct {
	envConfig *config.Configuration
	svcStore  *ServiceStore
}

func NewService(envConfig *config.Configuration, svcStore *ServiceStore) Service {
	return &governanceService{
		envConfig: envConfig,
		svcStore:  svcStore,
	}
}

func (s *governanceService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

func (s *governanceService) Login(ctx context.Context, username, password string) (string, string, *bson.M, int, error) {
	fail := func(err error) (string, string, *bson.M, int, error) {
		return "", "", nil, 0, err
	}

	if username == "" || password == "" {
		return fail(errors.ErrMissingCreds)
	}
	// TODO: implement proper credentials check
	if password != "password" {
		return fail(errors.ErrInvalidCreds)
	}

	// TODO: get user id and role from db
	id := "0334ded6-047f-4c90-9e7f-05d84c94b530"
	role := username

	// create tokens
	now := time.Now()
	tokenAccess, err := auth.CreateToken(id, role, s.envConfig.SecretKey, auth.AccessToken, now)
	if err != nil {
		return fail(err)
	}
	tokenRefresh, err := auth.CreateToken(id, role, s.envConfig.SecretKey, auth.RefreshToken, now)
	if err != nil {
		return fail(err)
	}

	// fetch metadata
	mdata, err := s.fetchMetaData(ctx, id)
	if err != nil {
		return fail(err)
	}

	return tokenAccess, tokenRefresh, mdata, http.StatusOK, nil
}

func (s *governanceService) Logout(ctx context.Context, tokenAccess, tokenRefresh string, claimsAccess, claimsRefresh jwt.MapClaims) (int, error) {
	// add access-token to blacklist
	err := auth.BlacklistToken(ctx, s.svcStore.Redis, tokenAccess, claimsAccess)
	if err != nil {
		return 0, err
	}

	// add refresh-token to blacklist
	err = auth.BlacklistToken(ctx, s.svcStore.Redis, tokenRefresh, claimsRefresh)
	if err != nil {
		return 0, err
	}

	return http.StatusNoContent, nil
}

func (s *governanceService) Refresh(ctx context.Context, curTokenRefresh string, claimsRefresh jwt.MapClaims) (string, string, *bson.M, int, error) {
	fail := func(err error) (string, string, *bson.M, int, error) {
		return "", "", nil, 0, err
	}

	// add the current refresh-token to blacklist
	err := auth.BlacklistToken(ctx, s.svcStore.Redis, curTokenRefresh, claimsRefresh)
	if err != nil {
		return fail(err)
	}

	// TODO: blacklist all access tokens issued with this refresh token

	// create new tokens
	id, _ := claimsRefresh["sub"].(string)
	role, _ := claimsRefresh["aud"].(string)
	now := time.Now()
	tokenAccess, err := auth.CreateToken(id, role, s.envConfig.SecretKey, auth.AccessToken, now)
	if err != nil {
		return fail(err)
	}
	tokenRefresh, err := auth.CreateToken(id, role, s.envConfig.SecretKey, auth.RefreshToken, now)
	if err != nil {
		return fail(err)
	}

	// fetch metadata
	mdata, err := s.fetchMetaData(ctx, id)
	if err != nil {
		return fail(err)
	}

	return tokenAccess, tokenRefresh, mdata, http.StatusOK, nil
}

func (s *governanceService) Authenticate(ctx context.Context, token string) (jwt.MapClaims, int, error) {
	fail := func(err error) (jwt.MapClaims, int, error) {
		svcErr, _ := err.(*errors.ServiceError)
		return nil, svcErr.HTTPCode, err
	}

	claims, err := auth.VerifyToken(ctx, s.svcStore.Redis, token, s.envConfig.SecretKey, auth.AccessToken)
	if err != nil {
		return fail(err)
	}

	return claims, http.StatusOK, nil
}

func (s *governanceService) Authorize(ctx context.Context, token, obj, act string) (bool, jwt.MapClaims, int, error) {
	fail := func(err error) (bool, jwt.MapClaims, int, error) {
		svcErr, _ := err.(*errors.ServiceError)
		return false, nil, svcErr.HTTPCode, err
	}

	claims, err := auth.VerifyToken(ctx, s.svcStore.Redis, token, s.envConfig.SecretKey, auth.AccessToken)
	if err != nil {
		return fail(err)
	}

	// Note: sub here refers to the subject in Casbin's model definition,
	// and not the sub in jwt claims
	sub, _ := claims["aud"].(string)

	eft, err := s.svcStore.Casbin.CheckPermission(sub, obj, act)
	if err != nil {
		return fail(err)
	}

	if !eft {
		return eft, claims, errors.ErrAccessDenied.HTTPCode, errors.ErrAccessDenied
	}

	return eft, claims, http.StatusOK, nil
}

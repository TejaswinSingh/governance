package governance

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	"github.com/go-kit/log"
	"github.com/golang-jwt/jwt/v5"
)

type governanceService struct {
	secretKey    string
	redisService *persist.RedisService
}

func NewService(secretKey string, redisService *persist.RedisService) Service {
	return &governanceService{secretKey: secretKey, redisService: redisService}
}

func (s *governanceService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

func (s *governanceService) Login(ctx context.Context, username, password string) (tokenAccess, tokenRefresh string, err error) {
	if username == "" || password == "" {
		return "", "", errors.ErrMissingCreds
	}
	// TODO: implement proper credentials check
	// if username == "admin" && password == "admin"
	if password == "password" {
		now := time.Now()
		tokenAccess, err = auth.CreateToken(username, s.secretKey, "access", now)
		if err != nil {
			return "", "", err
		}
		tokenRefresh, err = auth.CreateToken(username, s.secretKey, "refresh", now)
		if err != nil {
			return "", "", err
		}
		return tokenAccess, tokenRefresh, nil
	} else {
		return "", "", errors.ErrInvalidCreds
	}
}

func (s *governanceService) Logout(ctx context.Context, tokenAccess, tokenRefresh string, claimsAccess, claimsRefresh jwt.MapClaims) error {
	// add the access-token to blacklist
	err := auth.BlacklistToken(ctx, s.redisService, tokenAccess, claimsAccess)
	if err != nil {
		return err
	}

	// add the refresh-token to blacklist
	err = auth.BlacklistToken(ctx, s.redisService, tokenRefresh, claimsRefresh)
	if err != nil {
		return err
	}

	return nil
}

func (s *governanceService) Refresh(ctx context.Context, curTokenRefresh string, claimsRefresh jwt.MapClaims) (tokenAccess, tokenRefresh string, err error) {
	// add the current refresh-token to blacklist
	err = auth.BlacklistToken(ctx, s.redisService, curTokenRefresh, claimsRefresh)
	if err != nil {
		return "", "", err
	}

	// TODO: blacklist all access tokens issued with this refresh token

	// create new access and refresh tokens
	username, _ := claimsRefresh["sub"].(string)
	now := time.Now()
	tokenAccess, err = auth.CreateToken(username, s.secretKey, "access", now)
	if err != nil {
		return "", "", err
	}
	tokenRefresh, err = auth.CreateToken(username, s.secretKey, "refresh", now)
	if err != nil {
		return "", "", err
	}
	return tokenAccess, tokenRefresh, nil
}

func (s *governanceService) ValidateToken(ctx context.Context, token string) (int, error) {
	_, err := auth.VerifyToken(ctx, s.redisService, token, s.secretKey, "access")
	if err != nil {
		err, _ := err.(*errors.ServiceError)
		return err.HTTPCode, err
	}
	return http.StatusOK, nil
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}

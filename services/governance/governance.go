package governance

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/TejaswinSingh/version0/services/governance/auth"
	"github.com/TejaswinSingh/version0/services/governance/errors"
	"github.com/TejaswinSingh/version0/services/governance/redis"
	"github.com/go-kit/log"
	"github.com/golang-jwt/jwt/v5"
)

type governanceService struct {
	secretKey    string
	redisService *redis.RedisService
}

func NewService(secretKey string, redisService *redis.RedisService) Service {
	return &governanceService{secretKey: secretKey, redisService: redisService}
}

func (s *governanceService) ServiceStatus(_ context.Context) (int, *errors.ServiceError) {
	return http.StatusOK, nil
}

func (s *governanceService) Login(ctx context.Context, username, password string) (string, *errors.ServiceError) {
	if username == "" || password == "" {
		return "", errors.ErrMissingCreds
	}
	// TODO: implement proper credentials check
	if username == "admin" && password == "admin" {
		tokenString, err := auth.CreateToken(username, s.secretKey)
		if err != nil {
			return "", err
		}
		return tokenString, nil
	} else {
		return "", errors.ErrInvalidCreds
	}
}

func (s *governanceService) Logout(ctx context.Context) *errors.ServiceError {
	tokenString, ok := ctx.Value(TokenKey).(string)
	if !ok || tokenString == "" { // JWT auth middleware makes sure this never happens
		return nil
	}

	claims, ok := ctx.Value(ClaimsKey).(jwt.MapClaims) // same for this
	if !ok {
		return nil
	}

	expTime, ok := claims["exp"].(float64)
	if !ok {
		return nil
	}

	expirationDuration := time.Until(time.Unix(int64(expTime), 0))
	if expirationDuration <= 0 { // Token is already expired
		return nil
	}

	// add tokenString(JWT) to blacklist
	err := s.redisService.Set(ctx, tokenString, "blacklisted", expirationDuration)
	if err != nil {
		return err
	}

	return nil
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}

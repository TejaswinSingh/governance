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
	"go.mongodb.org/mongo-driver/v2/bson"
)

type governanceService struct {
	secretKey    string
	redisService *persist.RedisService
	mongoService *persist.MongoService
}

func NewService(secretKey string, redisService *persist.RedisService, mongoService *persist.MongoService) Service {
	return &governanceService{
		secretKey:    secretKey,
		redisService: redisService,
		mongoService: mongoService,
	}
}

func (s *governanceService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

func (s *governanceService) Login(ctx context.Context, username, password string) (tokenAccess, tokenRefresh string, mdata *bson.M, err error) {
	fail := func(err error) (string, string, *bson.M, error) {
		return "", "", nil, err
	}

	if username == "" || password == "" {
		return fail(errors.ErrMissingCreds)
	}
	// TODO: implement proper credentials check
	if password != "password" {
		return fail(errors.ErrInvalidCreds)
	}

	// create tokens
	now := time.Now()
	tokenAccess, err = auth.CreateToken(username, s.secretKey, "access", now)
	if err != nil {
		return fail(err)
	}
	tokenRefresh, err = auth.CreateToken(username, s.secretKey, "refresh", now)
	if err != nil {
		return fail(err)
	}

	// fetch metadata
	id := "f3f8be14-517f-4514-b712-f61046aad21b" // TODO: get user id from db
	mdata, err = s.fetchMetaData(ctx, id)
	if err != nil {
		return fail(err)
	}

	return tokenAccess, tokenRefresh, mdata, nil
}

func (s *governanceService) Logout(ctx context.Context, tokenAccess, tokenRefresh string, claimsAccess, claimsRefresh jwt.MapClaims) error {
	// add access-token to blacklist
	err := auth.BlacklistToken(ctx, s.redisService, tokenAccess, claimsAccess)
	if err != nil {
		return err
	}

	// add refresh-token to blacklist
	err = auth.BlacklistToken(ctx, s.redisService, tokenRefresh, claimsRefresh)
	if err != nil {
		return err
	}

	return nil
}

func (s *governanceService) Refresh(ctx context.Context, curTokenRefresh string, claimsRefresh jwt.MapClaims) (tokenAccess, tokenRefresh string, mdata *bson.M, err error) {
	fail := func(err error) (string, string, *bson.M, error) {
		return "", "", nil, err
	}

	// add the current refresh-token to blacklist
	err = auth.BlacklistToken(ctx, s.redisService, curTokenRefresh, claimsRefresh)
	if err != nil {
		return fail(err)
	}

	// TODO: blacklist all access tokens issued with this refresh token

	// create new tokens
	username, _ := claimsRefresh["sub"].(string)
	now := time.Now()
	tokenAccess, err = auth.CreateToken(username, s.secretKey, "access", now)
	if err != nil {
		return fail(err)
	}
	tokenRefresh, err = auth.CreateToken(username, s.secretKey, "refresh", now)
	if err != nil {
		return fail(err)
	}

	// fetch metadata
	id := "f3f8be14-517f-4514-b712-f61046aad21b" // TODO: get user id from claims or db
	mdata, err = s.fetchMetaData(ctx, id)
	if err != nil {
		return fail(err)
	}

	return tokenAccess, tokenRefresh, mdata, nil
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

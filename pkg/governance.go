package governance

import (
	"context"
	"net/http"
	"net/mail"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/config"
	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/twilio"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type governanceService struct {
	envConfig    *config.Configuration
	svcStore     *ServiceStore
	twilioClient twilio.TwilioClient
}

func NewService(envConfig *config.Configuration, svcStore *ServiceStore, twC twilio.TwilioClient) Service {
	return &governanceService{
		envConfig:    envConfig,
		svcStore:     svcStore,
		twilioClient: twC,
	}
}

func (s *governanceService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

func (s *governanceService) GetSchoolAdminToken(ctx context.Context, email string) (token string, status int, err error) {

	// validate email
	if _, err := mail.ParseAddress(email); err != nil {
		return "", 0, errors.ErrBadRequest
	}

	// make uuid from email
	id := UUIDv5FromPhone(email, s.envConfig.NamespaceUUID)

	// check user exists
	userDoc, err := s.getUser(ctx, id)
	if err != nil {
		return "", 0, err
	}

	role := userDoc["role"].(string)
	token, err = auth.CreateToken(id.String(), auth.Role(role), s.envConfig.SecretKey, auth.AccessToken, time.Now())
	if err != nil {
		return "", 0, err
	}
	// encrypt token
	token, err = auth.EncryptJWT(token, s.envConfig.SecretKey)
	if err != nil {
		return "", 0, err
	}

	return token, http.StatusOK, nil
}

func (s *governanceService) GetSchoolAdminMetadata(ctx context.Context, claims jwt.MapClaims) (*bson.M, int, error) {

	id, _ := claims["sub"].(string)

	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, 0, errors.ErrInternalServer.WithInternalErr(err)
	}

	mdata, err := s.fetchSchoolAdminMetaData(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return &mdata, http.StatusOK, nil
}

func (s *governanceService) SendOTP(ctx context.Context, countryCode, phone string) (int, error) {
	phone = countryCode + phone

	// make uuid from phone
	id := UUIDv5FromPhone(phone, s.envConfig.NamespaceUUID)

	// check user exists
	if _, err := s.getUser(ctx, id); err != nil {
		return 0, err
	}

	// send otp
	if err := s.twilioClient.SendOTP(phone); err != nil {
		return 0, err
	}

	return http.StatusOK, nil
}

func (s *governanceService) Login(ctx context.Context, countryCode, phone, otp string) (string, string, *bson.M, int, error) {
	fail := func(err error) (string, string, *bson.M, int, error) {
		return "", "", nil, 0, err
	}

	phone = countryCode + phone

	// otp-verification
	if err := s.twilioClient.VerifyOTP(phone, otp); err != nil {
		return fail(err)
	}

	// make uuid from phone
	id := UUIDv5FromPhone(phone, s.envConfig.NamespaceUUID)

	// fetch metadata
	mdata, err := s.fetchParentMetaData(ctx, id)
	if err != nil {
		return fail(err)
	}

	role := mdata["role"].(string)

	// create tokens
	now := time.Now()
	tokenAccess, err := auth.CreateToken(id.String(), auth.Role(role), s.envConfig.SecretKey, auth.AccessToken, now)
	if err != nil {
		return fail(err)
	}
	tokenRefresh, err := auth.CreateToken(id.String(), auth.Role(role), s.envConfig.SecretKey, auth.RefreshToken, now)
	if err != nil {
		return fail(err)
	}

	return tokenAccess, tokenRefresh, &mdata, http.StatusOK, nil
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

	id, _ := claimsRefresh["sub"].(string)
	role, _ := claimsRefresh["role"].(string)

	userID, err := uuid.Parse(id)
	if err != nil {
		return fail(errors.ErrInvalidClaims)
	}

	// create new tokens
	now := time.Now()
	tokenAccess, err := auth.CreateToken(userID.String(), auth.Role(role), s.envConfig.SecretKey, auth.AccessToken, now)
	if err != nil {
		return fail(err)
	}
	tokenRefresh, err := auth.CreateToken(userID.String(), auth.Role(role), s.envConfig.SecretKey, auth.RefreshToken, now)
	if err != nil {
		return fail(err)
	}

	// fetch metadata
	mdata, err := s.fetchParentMetaData(ctx, userID)
	if err != nil {
		return fail(err)
	}

	return tokenAccess, tokenRefresh, &mdata, http.StatusOK, nil
}

func (s *governanceService) Authenticate(ctx context.Context, token string) (jwt.MapClaims, int, error) {
	fail := func(err error) (jwt.MapClaims, int, error) {
		svcErr, _ := err.(*errors.ServiceError)
		return nil, svcErr.HTTPCode, err
	}

	tk, err := auth.DecodeAndDecryptJWT(token, s.envConfig.SecretKey)
	if err != nil {
		return fail(errors.ErrUnauthorized)
	}

	claims, err := auth.VerifyToken(ctx, s.svcStore.Redis, string(tk), s.envConfig.SecretKey, auth.AccessToken)
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

	tk, err := auth.DecodeAndDecryptJWT(token, s.envConfig.SecretKey)
	if err != nil {
		return fail(errors.ErrUnauthorized)
	}

	claims, err := auth.VerifyToken(ctx, s.svcStore.Redis, string(tk), s.envConfig.SecretKey, auth.AccessToken)
	if err != nil {
		return fail(err)
	}

	// Note: sub here refers to the subject in Casbin's model definition,
	// and not the sub in jwt claims
	sub, _ := claims["role"].(string)

	eft, err := s.svcStore.Casbin.CheckPermission(sub, obj, act)
	if err != nil {
		return fail(err)
	}

	if !eft {
		return eft, claims, errors.ErrAccessDenied.HTTPCode, errors.ErrAccessDenied
	}

	return eft, claims, http.StatusOK, nil
}

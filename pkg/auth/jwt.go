package auth

import (
	"context"
	"os"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	"github.com/go-kit/log"
	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenExpiry  = time.Hour * 1       // 1 hour
	refreshTokenExpiry = time.Hour * 24 * 15 // 15 days
)

func CreateToken(id, role, secretKey, tokenType string, now time.Time) (string, error) {
	var exp time.Duration
	if tokenType == "access" {
		exp = accessTokenExpiry
	} else { // refresh
		exp = refreshTokenExpiry
	}
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type": tokenType,             // Type of token
		"sub":  id,                    // Subject (user identifier)
		"iss":  "resolyte.governance", // Issuer
		"aud":  role,                  // Audience (user role)
		"exp":  now.Add(exp).Unix(),   // Expiration time
		"iat":  now.Unix(),            // Issued at
	})

	tokenString, err := claims.SignedString([]byte(secretKey))
	if err != nil {
		return "", errors.ErrCreateJWT.SetInternalErr(err)
	}

	return tokenString, nil
}

func ExtractToken(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return authHeader
}

func VerifyToken(ctx context.Context, redisService *persist.RedisService, tokenString, secretKey, tokenType string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, errors.ErrInvalidToken
	}

	// validate signing method
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.ErrInvalidSignMethod
		}
		return []byte(secretKey), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil || !token.Valid {
		return nil, errors.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.ErrInvalidClaims
	}

	// validate token expiration
	expTime, ok := claims["exp"].(float64) // JWT stores exp as float64
	if !ok {
		return nil, errors.ErrInvalidClaims
	}
	if time.Now().Unix() > int64(expTime) {
		return nil, errors.ErrInvalidToken
	}

	// validate token type
	tokenTypeClaim, ok := claims["type"].(string)
	if !ok || tokenTypeClaim != tokenType {
		return nil, errors.ErrInvalidClaims
	}

	// validate token subject
	// TODO: more thorough validation of subject by either
	// querying mongodb or the query-engine service
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, errors.ErrInvalidClaims
	}

	// validate aud(role)
	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return nil, errors.ErrInvalidClaims
	}

	// validate that token is not blacklisted
	val, err := redisService.Get(ctx, tokenString)
	if err != nil && err != errors.ErrRedisKeyNotExist {
		return nil, err
	}
	if err == nil && val == "blacklisted" {
		return nil, errors.ErrInvalidToken
	}

	return claims, nil
}

// compares two JWT claims
func CompareJWTClaims(c1, c2 jwt.MapClaims, keys ...string) bool {
	for _, key := range keys {
		v1, ok1 := c1[key].(string)
		v2, ok2 := c2[key].(string)
		if !ok1 || !ok2 || v1 != v2 {
			return false
		}
	}
	return true
}

func BlacklistToken(ctx context.Context, redisService *persist.RedisService, token string, claims jwt.MapClaims) error {
	expTime, ok := claims["exp"].(float64)
	if !ok {
		return errors.ErrInvalidClaims
	}
	expirationDuration := time.Until(time.Unix(int64(expTime), 0))
	err := redisService.Set(ctx, token, "blacklisted", expirationDuration)
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

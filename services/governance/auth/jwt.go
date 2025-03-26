package auth

import (
	"os"
	"time"

	"github.com/TejaswinSingh/version0/services/governance/errors"
	"github.com/go-kit/log"
	"github.com/golang-jwt/jwt/v5"
)

func CreateToken(username, secretKey string) (string, *errors.ServiceError) {
	now := time.Now()
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,                  // Subject (user identifier)
		"iss": "governance-service",      // Issuer
		"aud": getRole(username),         // Audience (user role)
		"exp": now.Add(time.Hour).Unix(), // Expiration time
		"iat": now.Unix(),                // Issued at
	})

	tokenString, err := claims.SignedString([]byte(secretKey))
	if err != nil {
		return "", errors.ErrCreateJWT
	}

	return tokenString, nil
}

func ExtractToken(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return authHeader
}

func VerifyToken(tokenString, secretKey string) (jwt.MapClaims, *errors.ServiceError) {
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

	expTime, ok := claims["exp"].(float64) // JWT stores exp as float64
	if !ok {
		return nil, errors.ErrInvalidClaims
	}
	// token expired
	if time.Now().Unix() > int64(expTime) {
		return nil, errors.ErrInvalidToken
	}

	return claims, nil
}

func getRole(username string) string {
	// TODO: implement proper role fetching
	if username == "admin" {
		return "admin"
	}
	return "employee"
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}

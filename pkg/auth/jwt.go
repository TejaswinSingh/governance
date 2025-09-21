package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/cryptoutil"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	"github.com/golang-jwt/jwt/v5"
)

type AuthToken string
type Role string

const (
	Domain                       string    = "abcdery.com"
	AccessToken                  AuthToken = "access_token"
	RefreshToken                 AuthToken = "refresh_token"
	RoleParent                   Role      = "parent"
	RoleDelegateParent           Role      = "delegate-parent"
	RoleSchoolAdmin              Role      = "school-admin"
	AccessTokenExpiry                      = time.Minute * 5      // 5 minutes
	SchoolAdminAccessTokenExpiry           = time.Hour * 24 * 365 // 1 year
	RefreshTokenExpiry                     = time.Hour * 24 * 15  // 15 days
)

type OtherClaims struct {
	Name string
	Val  any
}

func CreateToken(id string, role Role, secretKey []byte, tk AuthToken, now time.Time, ot *OtherClaims) (string, error) {
	var exp time.Duration

	if tk == AccessToken {
		if role == RoleSchoolAdmin {
			exp = SchoolAdminAccessTokenExpiry
		} else { // when role is either parent or delegate-parent
			exp = AccessTokenExpiry
		}
	} else { // refresh
		exp = RefreshTokenExpiry
	}

	mapClaims := jwt.MapClaims{
		"type": string(tk),          // Type of token
		"sub":  id,                  // Subject (user identifier)
		"iss":  "auth.abcdery.com",  // Issuer
		"aud":  "abcdery.users",     // Audience
		"role": role,                // user role
		"iat":  now.Unix(),          // Issued at
		"exp":  now.Add(exp).Unix(), // Expiration time
	}

	if ot != nil {
		mapClaims[ot.Name] = ot.Val
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mapClaims)

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", myerrors.ErrCreateJWT.WithInternalErr(err)
	}

	return tokenString, nil
}

func ExtractClaims(tokenStr string) (jwt.MapClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, myerrors.ErrInvalidToken.WithInternalErr(err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, myerrors.ErrInvalidClaims
}

func VerifyToken(ctx context.Context, redisService *persist.RedisService, tokenString string, secretKey []byte, tk AuthToken) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, myerrors.ErrInvalidToken
	}

	// validate signing method
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, myerrors.ErrInvalidSignMethod
		}
		return secretKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil || !token.Valid {
		return nil, myerrors.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, myerrors.ErrInvalidClaims
	}

	// validate token expiration
	expTime, ok := claims["exp"].(float64) // JWT stores exp as float64
	if !ok {
		return nil, myerrors.ErrInvalidClaims
	}
	if time.Now().Unix() > int64(expTime) {
		return nil, myerrors.ErrInvalidToken
	}

	// validate token type
	tokenTypeClaim, ok := claims["type"].(string)
	if !ok || tokenTypeClaim != string(tk) {
		return nil, myerrors.ErrInvalidClaims
	}

	// validate token subject
	// TODO: more thorough validation of subject by either
	// querying mongodb or the query-engine service
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, myerrors.ErrInvalidClaims
	}

	// validate aud
	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return nil, myerrors.ErrInvalidClaims
	}

	// validate role
	role, ok := claims["role"].(string)
	if !ok || role == "" {
		return nil, myerrors.ErrInvalidClaims
	}

	// validate that token is not blacklisted
	val, err := redisService.Get(ctx, tokenString)
	if err != nil && !errors.Is(err, myerrors.ErrRedisKeyNotExist) {
		return nil, err
	}
	if err == nil && val == "blacklisted" {
		return nil, myerrors.ErrInvalidToken
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
		return myerrors.ErrInvalidClaims
	}
	expirationDuration := time.Until(time.Unix(int64(expTime), 0))
	err := redisService.Set(ctx, token, "blacklisted", expirationDuration)
	if err != nil {
		return err
	}
	return nil
}

func SetAuthCookie(w http.ResponseWriter, secretKey []byte, token AuthToken, value string) {
	var (
		maxAge  int
		expires time.Time
		path    string
	)
	switch token {
	case AccessToken:
		maxAge = int(AccessTokenExpiry.Seconds())
		expires = time.Now().Add(AccessTokenExpiry)
		path = "/"
	case RefreshToken:
		maxAge = int(RefreshTokenExpiry.Seconds())
		expires = time.Now().Add(RefreshTokenExpiry)
		path = "/"
	}
	c := &http.Cookie{
		Name:     string(token),
		Domain:   Domain,
		Path:     path,
		Value:    value,
		MaxAge:   maxAge,
		Expires:  expires,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	WriteEncryptedCookie(w, c, secretKey)
}

func ClearAuthCookie(w http.ResponseWriter, token AuthToken) {
	var path string
	switch token {
	case AccessToken:
		path = "/"
	case RefreshToken:
		path = "/"
	}
	// no need to encrypt, since value is empty
	http.SetCookie(w, &http.Cookie{
		Name:     string(token),
		Domain:   Domain,
		Path:     path,
		Value:    "",
		MaxAge:   -1, // Marks cookie for immediate removal
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func DecodeAndDecryptJWT(token string, secretKey []byte) (string, error) {
	// base64 decode
	tk, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}

	// decrypt
	tk, err = cryptoutil.Decrypt(tk, secretKey)
	if err != nil {
		return "", err
	}
	return string(tk), nil
}

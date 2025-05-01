package endpoints

import (
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ServiceStatusRequest struct{}
type ServiceStatusResponse struct {
	Code int   `json:"status"`
	Err  error `json:"err,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginResponse struct {
	TokenAccess  string  `json:"-"`
	TokenRefresh string  `json:"-"`
	MetaData     *bson.M `json:"metadata,omitempty"`
	Err          error   `json:"err,omitempty"`
}

func (l LoginResponse) GetAccessToken() string  { return l.TokenAccess }
func (l LoginResponse) GetRefreshToken() string { return l.TokenRefresh }

type LogoutRequest struct {
	TokenAccess   string
	TokenRefresh  string
	ClaimsAccess  jwt.MapClaims
	ClaimsRefresh jwt.MapClaims
}
type LogoutResponse struct {
	Err error `json:"err,omitempty"`
}

type RefreshRequest struct {
	TokenRefresh  string
	ClaimsRefresh jwt.MapClaims
}
type RefreshResponse struct {
	TokenAccess  string  `json:"-"`
	TokenRefresh string  `json:"-"`
	MetaData     *bson.M `json:"metadata,omitempty"`
	Err          error   `json:"err,omitempty"`
}

func (r RefreshResponse) GetAccessToken() string  { return r.TokenAccess }
func (r RefreshResponse) GetRefreshToken() string { return r.TokenRefresh }

type ValidateTokenRequest struct {
	Token string
}
type ValidateTokenResponse struct {
	Code int
	Err  error
}

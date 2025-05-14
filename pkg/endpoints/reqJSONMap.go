package endpoints

import (
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ServiceStatusRequest struct{}
type ServiceStatusResponse struct {
	Status int `json:"status"`
}

func (r ServiceStatusResponse) GetStatus() int { return r.Status }

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginResponse struct {
	TokenAccess  string  `json:"-"`
	TokenRefresh string  `json:"-"`
	MetaData     *bson.M `json:"metadata,omitempty"`
	Status       int     `json:"status"`
}

func (l LoginResponse) GetAccessToken() string  { return l.TokenAccess }
func (l LoginResponse) GetRefreshToken() string { return l.TokenRefresh }
func (r LoginResponse) GetStatus() int          { return r.Status }

type LogoutRequest struct {
	TokenAccess   string
	TokenRefresh  string
	ClaimsAccess  jwt.MapClaims
	ClaimsRefresh jwt.MapClaims
}
type LogoutResponse struct {
	Status int `json:"-"`
}

func (r LogoutResponse) GetStatus() int { return r.Status }

type RefreshRequest struct {
	TokenRefresh  string
	ClaimsRefresh jwt.MapClaims
}
type RefreshResponse struct {
	TokenAccess  string  `json:"-"`
	TokenRefresh string  `json:"-"`
	MetaData     *bson.M `json:"metadata,omitempty"`
	Status       int     `json:"status"`
}

func (r RefreshResponse) GetAccessToken() string  { return r.TokenAccess }
func (r RefreshResponse) GetRefreshToken() string { return r.TokenRefresh }
func (r RefreshResponse) GetStatus() int          { return r.Status }

type ValidateTokenRequest struct {
	Token string
}
type ValidateTokenResponse struct {
	Status int
	Err    error
}

type AuthorizeRequest struct {
	Token string
	Obj   string
	Act   string
}
type AuthorizeResponse struct {
	Eft    bool
	Status int
	Err    error
}

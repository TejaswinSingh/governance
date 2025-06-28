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

type SendOTPRequest struct {
	CountryCode string `json:"country_code"`
	Phone       string `json:"phone"`
}
type SendOTPResponse struct {
	Status int `json:"status"`
}

func (r SendOTPResponse) GetStatus() int { return r.Status }

type LoginRequest struct {
	CountryCode string `json:"country_code"`
	Phone       string `json:"phone"`
	OTP         string `json:"otp"`
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

type AuthenticateRequest struct {
	Token string
}
type AuthenticateResponse struct {
	Claims jwt.MapClaims
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
	Claims jwt.MapClaims
	Status int
	Err    error
}

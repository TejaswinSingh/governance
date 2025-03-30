package endpoints

import (
	"context"
	"os"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/log"
	"github.com/golang-jwt/jwt/v5"
)

type Set struct {
	ServiceStatusEndpoint endpoint.Endpoint
	LoginEndpoint         endpoint.Endpoint
	LogoutEndpoint        endpoint.Endpoint
	RefreshEndpoint       endpoint.Endpoint
}

func NewEndpointSet(svc governance.Service) Set {
	return Set{
		ServiceStatusEndpoint: MakeServiceStatusEndpoint(svc),
		LoginEndpoint:         MakeLoginEndpoint(svc),
		LogoutEndpoint:        MakeLogoutEndpoint(svc),
		RefreshEndpoint:       MakeRefreshEndpoint(svc),
	}
}

func MakeServiceStatusEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		_ = request.(ServiceStatusRequest)
		code, err := svc.ServiceStatus(ctx)
		if err != nil {
			return err, nil
		}
		return ServiceStatusResponse{Code: code, Err: nil}, nil
	}
}

func MakeLoginEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(LoginRequest)
		tokenAccess, tokenRefresh, err := svc.Login(ctx, req.Username, req.Password)
		if err != nil {
			return err, nil
		}
		return LoginResponse{
			TokenAccess:  tokenAccess,
			TokenRefresh: tokenRefresh,
			Err:          nil}, nil
	}
}

func MakeLogoutEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		_ = request.(LogoutRequest)
		// get values extracted by the auth middleware
		tokenAccess, _ := ctx.Value(governance.AccessTokenKey).(string)
		tokenRefresh, _ := ctx.Value(governance.RefreshTokenKey).(string)
		accessClaims, _ := ctx.Value(governance.AccessClaimsKey).(jwt.MapClaims)
		refreshClaims, _ := ctx.Value(governance.RefreshClaimsKey).(jwt.MapClaims)
		err := svc.Logout(ctx, tokenAccess, tokenRefresh, accessClaims, refreshClaims)
		if err != nil {
			return err, nil
		}
		return LogoutResponse{Err: nil}, nil
	}
}

func MakeRefreshEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		_ = request.(RefreshRequest)
		token, _ := ctx.Value(governance.RefreshTokenKey).(string)
		claims, _ := ctx.Value(governance.RefreshClaimsKey).(jwt.MapClaims)
		tokenAccess, tokenRefresh, err := svc.Refresh(ctx, token, claims)
		if err != nil {
			return err, nil
		}
		return RefreshResponse{
			TokenAccess:  tokenAccess,
			TokenRefresh: tokenRefresh,
			Err:          nil}, nil
	}
}

// func (s *Set) ServiceStatus(ctx context.Context) (int, error) {
// 	resp, err := s.ServiceStatusEndpoint(ctx, ServiceStatusRequest{})
// 	svcStatusResp := resp.(ServiceStatusResponse)
// 	if err != nil {
// 		return svcStatusResp.Code, err
// 	}
// 	if svcStatusResp.Err != "" {
// 		return svcStatusResp.Code, errors.New(svcStatusResp.Err)
// 	}
// 	return svcStatusResp.Code, nil
// }

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}

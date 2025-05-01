package endpoints

import (
	"context"
	"os"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/log"
)

type Set struct {
	ServiceStatusEndpoint endpoint.Endpoint
	LoginEndpoint         endpoint.Endpoint
	LogoutEndpoint        endpoint.Endpoint
	RefreshEndpoint       endpoint.Endpoint
	ValidateTokenEndpoint endpoint.Endpoint
}

func NewEndpointSet(svc governance.Service) Set {
	return Set{
		ServiceStatusEndpoint: MakeServiceStatusEndpoint(svc),
		LoginEndpoint:         MakeLoginEndpoint(svc),
		LogoutEndpoint:        MakeLogoutEndpoint(svc),
		RefreshEndpoint:       MakeRefreshEndpoint(svc),
		ValidateTokenEndpoint: MakeValidateTokenEndpoint(svc),
	}
}

func MakeServiceStatusEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		_ = request.(ServiceStatusRequest)
		code, err := svc.ServiceStatus(ctx)
		if err != nil {
			return nil, err
		}
		return ServiceStatusResponse{Code: code, Err: nil}, nil
	}
}

func MakeLoginEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(LoginRequest)
		tokenAccess, tokenRefresh, mdata, err := svc.Login(ctx, req.Username, req.Password)
		if err != nil {
			return nil, err
		}
		return LoginResponse{
			TokenAccess:  tokenAccess,
			TokenRefresh: tokenRefresh,
			MetaData:     mdata,
			Err:          nil}, nil
	}
}

func MakeLogoutEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(LogoutRequest)
		err := svc.Logout(ctx, req.TokenAccess, req.TokenRefresh, req.ClaimsAccess, req.ClaimsRefresh)
		if err != nil {
			return nil, err
		}
		return LogoutResponse{Err: nil}, nil
	}
}

func MakeRefreshEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(RefreshRequest)
		tokenAccess, tokenRefresh, mdata, err := svc.Refresh(ctx, req.TokenRefresh, req.ClaimsRefresh)
		if err != nil {
			return nil, err
		}
		return RefreshResponse{
			TokenAccess:  tokenAccess,
			TokenRefresh: tokenRefresh,
			MetaData:     mdata,
			Err:          nil}, nil
	}
}

func MakeValidateTokenEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(ValidateTokenRequest)
		code, err := svc.ValidateToken(ctx, req.Token)
		// for grpc-only endpoints, pass
		// errors inside the response struct
		return ValidateTokenResponse{
			Code: code,
			Err:  err}, nil
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

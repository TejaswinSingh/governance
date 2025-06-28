package endpoints

import (
	"context"
	"errors"

	governance "github.com/Resolyte-official/service-governance/pkg"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

type Set struct {
	// http endpoints
	ServiceStatusEndpoint endpoint.Endpoint
	SendOTPEndpoint       endpoint.Endpoint
	LoginEndpoint         endpoint.Endpoint
	LogoutEndpoint        endpoint.Endpoint
	RefreshEndpoint       endpoint.Endpoint
	// grpc endpoints
	AuthenticateEndpoint endpoint.Endpoint
	AuthorizeEndpoint    endpoint.Endpoint
}

func NewEndpointSet(svc governance.Service) Set {
	return Set{
		ServiceStatusEndpoint: MakeServiceStatusEndpoint(svc),
		SendOTPEndpoint:       MakeSendOTPEndpoint(svc),
		LoginEndpoint:         MakeLoginEndpoint(svc),
		LogoutEndpoint:        MakeLogoutEndpoint(svc),
		RefreshEndpoint:       MakeRefreshEndpoint(svc),
		AuthenticateEndpoint:  MakeAuthenticateEndpoint(svc),
		AuthorizeEndpoint:     MakeAuthorizeEndpoint(svc),
	}
}

func MakeServiceStatusEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		_ = request.(ServiceStatusRequest)
		status, err := svc.ServiceStatus(ctx)
		if err != nil {
			return nil, err
		}
		return ServiceStatusResponse{Status: status}, nil
	}
}

func MakeSendOTPEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(SendOTPRequest)
		status, err := svc.SendOTP(ctx, req.CountryCode, req.Phone)
		if err != nil {
			return nil, err
		}
		return SendOTPResponse{Status: status}, nil
	}
}

func MakeLoginEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(LoginRequest)
		tokenAccess, tokenRefresh, mdata, status, err := svc.Login(ctx, req.CountryCode, req.Phone, req.OTP)
		if err != nil {
			return nil, err
		}
		return LoginResponse{
			TokenAccess:  tokenAccess,
			TokenRefresh: tokenRefresh,
			MetaData:     mdata,
			Status:       status,
		}, nil
	}
}

func MakeLogoutEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(LogoutRequest)
		status, err := svc.Logout(ctx, req.TokenAccess, req.TokenRefresh, req.ClaimsAccess, req.ClaimsRefresh)
		if err != nil {
			return nil, err
		}
		return LogoutResponse{Status: status}, nil
	}
}

func MakeRefreshEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(RefreshRequest)
		tokenAccess, tokenRefresh, mdata, status, err := svc.Refresh(ctx, req.TokenRefresh, req.ClaimsRefresh)
		if err != nil {
			return nil, err
		}
		return RefreshResponse{
			TokenAccess:  tokenAccess,
			TokenRefresh: tokenRefresh,
			MetaData:     mdata,
			Status:       status,
		}, nil
	}
}

func MakeAuthenticateEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(AuthenticateRequest)
		claims, status, err := svc.Authenticate(ctx, req.Token)
		// fatal errors
		if errors.Is(err, myerrors.ErrRedis) || errors.Is(err, myerrors.ErrInternalServer) {
			return nil, err
		}
		// for grpc-only endpoints, pass non-fatal
		// errors inside the response
		return AuthenticateResponse{
			Claims: claims,
			Status: status,
			Err:    err,
		}, nil
	}
}

func MakeAuthorizeEndpoint(svc governance.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(AuthorizeRequest)
		eft, claims, status, err := svc.Authorize(ctx, req.Token, req.Obj, req.Act)
		// fatal errors
		if errors.Is(err, myerrors.ErrRedis) || errors.Is(err, myerrors.ErrCasbinEnforcement) || errors.Is(err, myerrors.ErrInternalServer) {
			return nil, err
		}
		return AuthorizeResponse{
			Eft:    eft,
			Claims: claims,
			Status: status,
			Err:    err,
		}, nil
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

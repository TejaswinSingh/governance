package endpoints

import (
	"context"
	"os"

	"github.com/TejaswinSingh/version0/services/governance"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/log"
)

type Set struct {
	ServiceStatusEndpoint endpoint.Endpoint
	LoginEndpoint         endpoint.Endpoint
}

func NewEndpointSet(svc governance.Service) Set {
	return Set{
		ServiceStatusEndpoint: MakeServiceStatusEndpoint(svc),
		LoginEndpoint:         MakeLoginEndpoint(svc),
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
		token, err := svc.Login(ctx, req.Username, req.Password)
		if err != nil {
			return err, nil
		}
		return LoginResponse{Token: token, Err: nil}, nil
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

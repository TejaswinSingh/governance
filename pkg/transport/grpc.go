package transport

import (
	"context"

	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	"github.com/Resolyte-official/service-governance/pkg/pb"
	grpctransport "github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct {
	pb.UnimplementedGovernanceServer
	serviceStatus grpctransport.Handler
	validateToken grpctransport.Handler
}

func NewGRPCServer(ep endpoints.Set) pb.GovernanceServer {
	return &grpcServer{
		serviceStatus: grpctransport.NewServer(
			ep.ServiceStatusEndpoint,
			decodeGRPCServiceStatusRequest,
			encodeGRPCServiceStatusResponse,
		),
		validateToken: grpctransport.NewServer(
			ep.ValidateTokenEndpoint,
			decodeGRPCValidateTokenRequest,
			encodeGRPCValidateTokenResponse,
		),
	}
}

func (g *grpcServer) ServiceStatus(ctx context.Context, r *pb.ServiceStatusRequest) (*pb.ServiceStatusResponse, error) {
	_, resp, err := g.serviceStatus.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.ServiceStatusResponse), nil
}

func (g *grpcServer) ValidateToken(ctx context.Context, r *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	_, resp, err := g.validateToken.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.ValidateTokenResponse), nil
}

func decodeGRPCServiceStatusRequest(ctx context.Context, grpcReq any) (any, error) {
	return endpoints.ServiceStatusRequest{}, nil
}

func decodeGRPCValidateTokenRequest(ctx context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*pb.ValidateTokenRequest)
	return endpoints.ValidateTokenRequest{Token: req.Token}, nil
}

func encodeGRPCServiceStatusResponse(ctx context.Context, grpcResp any) (any, error) {
	resp := grpcResp.(endpoints.ServiceStatusResponse)
	var err string
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.ServiceStatusResponse{
		Code: int64(resp.Code),
		Err:  err,
	}, nil
}

func encodeGRPCValidateTokenResponse(ctx context.Context, grpcResp any) (any, error) {
	resp := grpcResp.(endpoints.ValidateTokenResponse)
	var err string
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.ValidateTokenResponse{
		Code: int64(resp.Code),
		Err:  err,
	}, nil
}

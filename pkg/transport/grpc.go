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
}

func NewGRPCServer(ep endpoints.Set) pb.GovernanceServer {
	return &grpcServer{
		serviceStatus: grpctransport.NewServer(
			ep.ServiceStatusEndpoint,
			decodeGRPCServiceStatusRequest,
			encodeGRPCServiceStatusResponse,
		),
	}
}

func (g *grpcServer) ServiceStatus(ctx context.Context, r *pb.ServiceStatusRequest) (*pb.ServiceStatusResponse, error) {
	_, rep, err := g.serviceStatus.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.ServiceStatusResponse), nil
}

func decodeGRPCServiceStatusRequest(ctx context.Context, grpcReq any) (any, error) {
	return endpoints.ServiceStatusRequest{}, nil
}

func encodeGRPCServiceStatusResponse(ctx context.Context, grpcResp any) (any, error) {
	reply := grpcResp.(endpoints.ServiceStatusResponse)
	var err string
	if reply.Err != nil {
		err = reply.Err.Error()
	}
	return &pb.ServiceStatusResponse{
		Code: int64(reply.Code),
		Err:  err,
	}, nil
}

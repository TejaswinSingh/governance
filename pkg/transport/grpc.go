package transport

import (
	"context"

	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/pb"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	pb.UnimplementedGovernanceServer
	serviceStatus grpctransport.Handler
	validateToken grpctransport.Handler
	authorize     grpctransport.Handler
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
		authorize: grpctransport.NewServer(
			ep.AuthorizeEndpoint,
			decodeGRPCAuthorizeRequest,
			encodeGRPCAuthorizeResponse,
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

func (g *grpcServer) Authorize(ctx context.Context, r *pb.AuthorizeRequest) (*pb.AuthorizeResponse, error) {
	_, resp, err := g.authorize.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthorizeResponse), nil
}

func decodeGRPCServiceStatusRequest(ctx context.Context, grpcReq any) (any, error) {
	return endpoints.ServiceStatusRequest{}, nil
}

func decodeGRPCValidateTokenRequest(ctx context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*pb.ValidateTokenRequest)
	if req.Token == "" {
		return nil, toGRPCError(myerrors.ErrGRPCBadRequest)
	}
	return endpoints.ValidateTokenRequest{Token: req.Token}, nil
}

func decodeGRPCAuthorizeRequest(ctx context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*pb.AuthorizeRequest)
	if req.Token == "" || req.Act == "" || req.Obj == "" {
		return nil, toGRPCError(myerrors.ErrGRPCBadRequest)
	}
	return endpoints.AuthorizeRequest{
		Token: req.Token,
		Obj:   req.Obj,
		Act:   req.Act,
	}, nil
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

func encodeGRPCAuthorizeResponse(ctx context.Context, grpcResp any) (any, error) {
	resp := grpcResp.(endpoints.AuthorizeResponse)
	var err string
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.AuthorizeResponse{
		Eft:  resp.Eft,
		Code: int64(resp.Code),
		Err:  err,
	}, nil
}

func toGRPCError(err error) error {
	if svcErr, ok := err.(*myerrors.ServiceError); ok {
		switch svcErr.HTTPCode {
		case 400:
			return status.Error(codes.InvalidArgument, svcErr.Error())
		case 401:
			return status.Error(codes.Unauthenticated, svcErr.Error())
		case 403:
			return status.Error(codes.PermissionDenied, svcErr.Error())
		case 404:
			return status.Error(codes.NotFound, svcErr.Error())
		case 500:
			return status.Error(codes.Internal, svcErr.Error())
		default:
			return status.Error(codes.Unknown, svcErr.Error())
		}
	}
	return status.Error(codes.Unknown, err.Error())
}

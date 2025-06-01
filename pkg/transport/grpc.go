package transport

import (
	"context"

	"github.com/Resolyte-official/service-governance/pkg/config"
	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/Resolyte-official/service-governance/pkg/pb"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	pb.UnimplementedGovernanceServer
	serviceStatus grpctransport.Handler
	authenticate  grpctransport.Handler
	authorize     grpctransport.Handler
}

func NewGRPCServer(ep endpoints.Set, envConfig *config.Configuration, lw logging.LogWriter) pb.GovernanceServer {
	logger := logging.NewContextLogger(lw, "GrpcServer")
	return &grpcServer{
		serviceStatus: grpcLoggingMiddleware(envConfig, logger, "ServiceStatus", grpctransport.NewServer(
			ep.ServiceStatusEndpoint,
			decodeGRPCServiceStatusRequest,
			encodeGRPCServiceStatusResponse,
		)),
		authenticate: grpcLoggingMiddleware(envConfig, logger, "Authenticate", grpctransport.NewServer(
			ep.AuthenticateEndpoint,
			decodeGRPCAuthenticateRequest,
			encodeGRPCAuthenticateResponse,
		)),
		authorize: grpcLoggingMiddleware(envConfig, logger, "Authorize", grpctransport.NewServer(
			ep.AuthorizeEndpoint,
			decodeGRPCAuthorizeRequest,
			encodeGRPCAuthorizeResponse,
		)),
	}
}

func (g *grpcServer) ServiceStatus(ctx context.Context, r *pb.ServiceStatusRequest) (*pb.ServiceStatusResponse, error) {
	_, resp, err := g.serviceStatus.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.ServiceStatusResponse), nil
}

func (g *grpcServer) Authenticate(ctx context.Context, r *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
	_, resp, err := g.authenticate.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthenticateResponse), nil
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

func decodeGRPCAuthenticateRequest(ctx context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*pb.AuthenticateRequest)
	if req.Token == "" {
		return nil, myerrors.ErrGRPCBadRequest
	}
	return endpoints.AuthenticateRequest{Token: req.Token}, nil
}

func decodeGRPCAuthorizeRequest(ctx context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*pb.AuthorizeRequest)
	if req.Token == "" || req.Act == "" || req.Obj == "" {
		return nil, myerrors.ErrGRPCBadRequest
	}
	return endpoints.AuthorizeRequest{
		Token: req.Token,
		Obj:   req.Obj,
		Act:   req.Act,
	}, nil
}

func encodeGRPCServiceStatusResponse(ctx context.Context, grpcResp any) (any, error) {
	resp := grpcResp.(endpoints.ServiceStatusResponse)
	return &pb.ServiceStatusResponse{
		Status: int64(resp.Status),
	}, nil
}

func encodeGRPCAuthenticateResponse(ctx context.Context, grpcResp any) (any, error) {
	resp := grpcResp.(endpoints.AuthenticateResponse)
	var err string
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.AuthenticateResponse{
		Claims: toPBClaims(resp.Claims),
		Status: int64(resp.Status),
		Err:    err,
	}, nil
}

func encodeGRPCAuthorizeResponse(ctx context.Context, grpcResp any) (any, error) {
	resp := grpcResp.(endpoints.AuthorizeResponse)
	var err string
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.AuthorizeResponse{
		Eft:    resp.Eft,
		Claims: toPBClaims(resp.Claims),
		Status: int64(resp.Status),
		Err:    err,
	}, nil
}

func toGrpcError(err error) error {

	// Convert to standard gRPC error
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

package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

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

func loggingMiddleware(ctx context.Context, req any, rpc string, f grpctransport.Handler) (any, error) {
	start := time.Now().UTC()

	_, resp, err := f.ServeGRPC(ctx, req)

	logArgs := []interface{}{
		"TRANSPORT", "GRPC",
		"RPC", rpc,
	}

	defer func() {
		logArgs = append(logArgs, "DURATION", time.Since(start).String())
		logger.Log(logArgs...)
	}()

	if err != nil {
		var serviceErr *myerrors.ServiceError
		if errors.As(err, &serviceErr) {
			logArgs = append(logArgs, "STATUS", serviceErr.HTTPCode, "ERROR", serviceErr.DetailedError())
		} else {
			defErr := myerrors.ErrInternalServer
			logArgs = append(logArgs, "STATUS", defErr.HTTPCode, "ERROR", fmt.Sprintf("%s [Internal Error: %s]", defErr.Error(), err.Error()))
		}
		return nil, toGrpcError(err)
	}

	logArgs = append(logArgs, "STATUS", http.StatusOK)

	return resp, nil
}

func (g *grpcServer) ServiceStatus(ctx context.Context, r *pb.ServiceStatusRequest) (*pb.ServiceStatusResponse, error) {
	rpc := "ServiceStatus"
	resp, err := loggingMiddleware(ctx, r, rpc, g.serviceStatus)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.ServiceStatusResponse), nil
}

func (g *grpcServer) ValidateToken(ctx context.Context, r *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	rpc := "ValidateToken"
	resp, err := loggingMiddleware(ctx, r, rpc, g.validateToken)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.ValidateTokenResponse), nil
}

func (g *grpcServer) Authorize(ctx context.Context, r *pb.AuthorizeRequest) (*pb.AuthorizeResponse, error) {
	rpc := "Authorize"
	resp, err := loggingMiddleware(ctx, r, rpc, g.authorize)
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
		return nil, myerrors.ErrGRPCBadRequest
	}
	return endpoints.ValidateTokenRequest{Token: req.Token}, nil
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

func encodeGRPCValidateTokenResponse(ctx context.Context, grpcResp any) (any, error) {
	resp := grpcResp.(endpoints.ValidateTokenResponse)
	var err string
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.ValidateTokenResponse{
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

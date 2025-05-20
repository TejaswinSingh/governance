package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/auth"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/metadata"
)

type grpcHandlerFunc func(ctx context.Context, req any) (context.Context, any, error)

func (f grpcHandlerFunc) ServeGRPC(ctx context.Context, req any) (context.Context, any, error) {
	return f(ctx, req)
}

func grpcLoggingMiddleware(logger logging.Logger, rpc string, next grpctransport.Handler) grpctransport.Handler {
	return grpcHandlerFunc(func(ctx context.Context, req any) (context.Context, any, error) {
		start := time.Now()

		logE := &LogEntry{Transport: "GRPC", RPC: rpc}

		defer func() {
			logE.Duration = time.Since(start).String()
			logE.Log(logger)
		}()

		// log userID if request has a token
		if r, ok := req.(interface{ GetToken() string }); ok {
			token := r.GetToken()
			if claims, err := auth.ExtractClaims(token); err == nil {
				logE.UserID, _ = claims["sub"].(string)
			}
		}

		ctx, resp, err := next.ServeGRPC(ctx, req)

		// look for request ID in metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			requestIDs := md.Get("X-Request-ID")
			if len(requestIDs) > 0 {
				logE.RequestID = requestIDs[0]
			}
		}

		if err != nil {
			var serviceErr *myerrors.ServiceError
			if errors.As(err, &serviceErr) {
				logE.Status = serviceErr.HTTPCode
				logE.Error = serviceErr.DetailedError()
			} else {
				defErr := myerrors.ErrInternalServer
				logE.Status = defErr.HTTPCode
				logE.Error = fmt.Sprintf("%s [Internal Error: %s]", defErr.Error(), err.Error())
			}
			return ctx, nil, toGrpcError(err)
		}

		logE.Status = http.StatusOK
		return ctx, resp, nil
	})
}

package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/config"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/Resolyte-official/service-governance/pkg/metrics"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type grpcHandlerFunc func(ctx context.Context, req any) (context.Context, any, error)

func (f grpcHandlerFunc) ServeGRPC(ctx context.Context, req any) (context.Context, any, error) {
	return f(ctx, req)
}

func grpcTracingLoggingMetricsMiddleware(envConfig *config.Configuration, logger logging.Logger, metrics *metrics.Metrics, rpc string, next grpctransport.Handler) grpctransport.Handler {
	return grpcHandlerFunc(func(ctx context.Context, req any) (context.Context, any, error) {
		start := time.Now()

		logE := &LogEntry{Transport: "GRPC", RPC: rpc}

		defer func() {
			logE.Duration = time.Since(start).String()
			logE.Log(logger)

			// metrics
			labels := []string{
				"transport", logE.Transport,
				"path", "",
				"rpc", logE.RPC,
				"status", strconv.Itoa(logE.Status),
			}
			metrics.RequestLatency.With(labels...).Observe(time.Since(start).Seconds())
			metrics.RequestCount.With(labels...).Add(1)
		}()

		// log userID if request has a token
		if r, ok := req.(interface{ GetToken() string }); ok {
			if token, err := auth.DecodeAndDecryptJWT(r.GetToken(), envConfig.SecretKey); err == nil {
				if claims, err := auth.ExtractClaims(token); err == nil {
					logE.UserID, _ = claims["sub"].(string)
				}
			}
		}

		var (
			reqID string
			err   error
			resp  any
		)

		// look for request ID in metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			requestIDs := md.Get("X-Request-ID")
			if len(requestIDs) > 0 {
				reqID = requestIDs[0]
			}
		}
		if reqID == "" {
			err = myerrors.ErrMissingRequestID
		} else if uuid.Validate(reqID) != nil {
			err = myerrors.ErrInvalidRequestID
		} else {
			logE.RequestID = reqID
			ctx, resp, err = next.ServeGRPC(ctx, req)
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

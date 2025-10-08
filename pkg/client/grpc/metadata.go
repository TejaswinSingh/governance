package grpc

import (
	"context"
	"time"

	gen_proto "github.com/Resolyte-official/gen_proto/pkg/metadata"
	"github.com/Resolyte-official/service-governance/pkg/contextkeys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type MetadataClient interface {
	ServiceStatus(ctx context.Context) (*gen_proto.ServiceStatusResponse, error)
	DelegateAccess(ctx context.Context, r *gen_proto.DelegateAccessRequest) (*gen_proto.DelegateAccessResponse, error)
	RevokeAccess(ctx context.Context, r *gen_proto.RevokeAccessRequest) (*gen_proto.RevokeAccessResponse, error)
	Close() error
}

type metadataClient struct {
	conn       *grpc.ClientConn
	grpcClient gen_proto.MetadataClient
	timeout    time.Duration
}

const requestTimeout = 5 * time.Second

func NewMetadataClient(address string, creds credentials.TransportCredentials) (MetadataClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return &metadataClient{
		conn:       conn,
		grpcClient: gen_proto.NewMetadataClient(conn),
		timeout:    requestTimeout,
	}, nil
}

func (c *metadataClient) Close() error {
	return c.conn.Close()
}

func (c *metadataClient) contextWithMetadata(ctx context.Context) (context.Context, context.CancelFunc) {
	reqID, _ := ctx.Value(contextkeys.RequestIDKey).(string)
	md := metadata.New(map[string]string{
		"X-REQUEST-ID": reqID,
	})
	newCtx := metadata.NewOutgoingContext(ctx, md)
	return context.WithTimeout(newCtx, c.timeout)
}

func (c *metadataClient) ServiceStatus(ctx context.Context) (*gen_proto.ServiceStatusResponse, error) {
	ctx, cancel := c.contextWithMetadata(ctx)
	defer cancel()
	return c.grpcClient.ServiceStatus(ctx, &gen_proto.ServiceStatusRequest{})
}

func (c *metadataClient) DelegateAccess(ctx context.Context, r *gen_proto.DelegateAccessRequest) (*gen_proto.DelegateAccessResponse, error) {
	ctx, cancel := c.contextWithMetadata(ctx)
	defer cancel()
	return c.grpcClient.DelegateAccess(ctx, r)
}

func (c *metadataClient) RevokeAccess(ctx context.Context, r *gen_proto.RevokeAccessRequest) (*gen_proto.RevokeAccessResponse, error) {
	ctx, cancel := c.contextWithMetadata(ctx)
	defer cancel()
	return c.grpcClient.RevokeAccess(ctx, r)
}

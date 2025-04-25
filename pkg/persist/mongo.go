package persist

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoService struct {
	client *mongo.Client
}

func NewMongoService(uri string) (*MongoService, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}
	return &MongoService{
		client: client,
	}, nil
}

func (m *MongoService) Ping(ctx context.Context) error {
	err := m.client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (m *MongoService) Disconnect(ctx context.Context) error {
	err := m.client.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

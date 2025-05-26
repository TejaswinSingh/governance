package persist

import (
	"context"
	"errors"
	"fmt"
	"time"

	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoConfiguration struct {
	Host     string
	Port     string
	Username string
	Password string
}

func CreateMongoURI(config MongoConfiguration) string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%s/", config.Username, config.Password, config.Host, config.Port)
}

type MongoService struct {
	client *mongo.Client
}

func NewMongoService(uri string) (*MongoService, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().SetTimeout(5 * time.Second).ApplyURI(uri).SetServerAPIOptions(serverAPI)
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

func (m *MongoService) GetCollection(db, coll string) *mongo.Collection {
	return m.client.Database(db).Collection(coll)
}

func (m *MongoService) FindOne(ctx context.Context, coll *mongo.Collection, filter any) (*mongo.SingleResult, error) {
	res := coll.FindOne(ctx, filter)
	if err := res.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
		return nil, myerrors.ErrMongo.SetInternalErr(err)
	}
	return res, nil
}

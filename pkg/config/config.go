package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/Resolyte-official/service-governance/pkg/persist"
)

// Configuration .
type Configuration struct {
	Env        GoEnv
	HttpMode   HttpMode
	HttpPort   string
	GrpcPort   string
	SecretKey  string
	CorsOrigin string
	Redis      persist.RedisConfiguration
	Mongo      persist.MongoConfiguration
}

const (
	defaultCORSOrigin    = "http://127.0.0.1:5500"
	defaultHTTPPort      = "12001"
	defaultGRPCPort      = "9090"
	defaultRedisHost     = "redis"
	defaultRedisPort     = "6379"
	defaultRedisDatabase = 0
	defaultMongoHost     = "mongo"
	defaultMongoPort     = "27017"
)

// NewEnvironmentConfiguration creates a configuration populated from os environment variables.
func NewEnvironmentConfiguration() Configuration {

	var httpMode HttpMode
	if envBool("USE_HTTPS", false) {
		httpMode = Https
	}

	redisDB := envInt("REDIS_DATABASE_GOVERNANCE", defaultRedisDatabase)
	if redisDB < 0 || redisDB > 15 {
		panic(errors.New("REDIS_DATABASE_GOVERNANCE must be an integer between 0 and 15, included"))
	}

	return Configuration{
		Env:        ValidateEnv(GoEnv(os.Getenv("ENV"))),
		HttpMode:   ValidateHttpMode(httpMode),
		HttpPort:   envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort),
		GrpcPort:   envString("GOVERNANCE_GRPC_PORT", defaultGRPCPort),
		SecretKey:  requireEnv("SECRET_KEY"),
		CorsOrigin: envString("CORS_ORIGIN", defaultCORSOrigin),
		Redis: persist.RedisConfiguration{
			Host:     envString("REDIS_HOST", defaultRedisHost),
			Port:     envString("REDIS_PORT", defaultRedisPort),
			Password: requireEnv("REDIS_PASSWORD"),
			Database: redisDB,
		},
		Mongo: persist.MongoConfiguration{
			Host:     envString("MONGO_HOST", defaultMongoHost),
			Port:     envString("MONGO_PORT", defaultMongoPort),
			Username: requireEnv("MONGODB_INITDB_ROOT_USERNAME"),
			Password: requireEnv("MONGODB_INITDB_ROOT_PASSWORD"),
		},
	}
}

func envString(key, fallback string) string {
	e, ok := os.LookupEnv(key)
	if e == "" || !ok {
		return fallback
	}
	return e
}

func envBool(key string, fallback bool) bool {
	val, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return val
}

func envInt(key string, fallback int) int {
	val, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return val
}

func requireEnv(env string) string {
	e, ok := os.LookupEnv(env)
	if e == "" || !ok {
		panic(fmt.Errorf("invalid environment configuration: %s is required", env))
	}
	return e
}

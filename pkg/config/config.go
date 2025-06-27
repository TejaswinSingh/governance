package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Resolyte-official/service-governance/pkg/persist"
)

// Configuration .
type Configuration struct {
	Env                    GoEnv
	HttpMode               HttpMode
	HttpPort               string
	GrpcPort               string
	SecretKey              []byte
	CorsOrigins            []string
	TwilioApiKey           string
	TwilioAccountSid       string
	TwilioApiSecret        string
	TwilioVerifyServiceSid string
	Redis                  persist.RedisConfiguration
	Mongo                  persist.MongoConfiguration
}

const (
	defaultCORSOrigin    = ""
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

	secretKey := []byte(requireEnv("SECRET_KEY"))
	if len(secretKey) < 32 {
		panic(fmt.Errorf("SECRET_KEY must be at least 32 bytes, got %d", len(secretKey)))
	}

	redisDB := envInt("REDIS_DATABASE_GOVERNANCE", defaultRedisDatabase)
	if redisDB < 0 || redisDB > 15 {
		panic(errors.New("REDIS_DATABASE_GOVERNANCE must be an integer between 0 and 15, included"))
	}

	return Configuration{
		Env:                    ValidateEnv(GoEnv(os.Getenv("ENV"))),
		HttpMode:               ValidateHttpMode(httpMode),
		HttpPort:               envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort),
		GrpcPort:               envString("GOVERNANCE_GRPC_PORT", defaultGRPCPort),
		SecretKey:              secretKey,
		CorsOrigins:            parseStringsEnv("CORS_ORIGINS", []string{defaultCORSOrigin}),
		TwilioApiKey:           requireEnv("TWILIO_API_KEY"),
		TwilioAccountSid:       requireEnv("TWILIO_ACCOUNT_SID"),
		TwilioApiSecret:        requireEnv("TWILIO_API_SECRET"),
		TwilioVerifyServiceSid: requireEnv("TWILIO_VERIFY_SERVICE_SID"),
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
	val := strings.TrimSpace(e)
	if !ok || val == "" {
		return fallback
	}
	return val
}

func envBool(key string, fallback bool) bool {
	val, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return val
}

func envInt(key string, fallback int) int {
	val, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return val
}

func requireEnv(key string) string {
	val, ok := os.LookupEnv(key)
	val = strings.TrimSpace(val)
	if !ok || val == "" {
		panic(fmt.Errorf("invalid environment configuration: %s is required", key))
	}
	return val
}

func parseStringsEnv(key string, fallback []string) []string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	parts := strings.Split(val, ",")
	var strs []string
	for _, o := range parts {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			strs = append(strs, trimmed)
		}
	}
	return strs
}

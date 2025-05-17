package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"
	"time"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	"github.com/Resolyte-official/service-governance/pkg/pb"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	"github.com/Resolyte-official/service-governance/pkg/transport"
	"google.golang.org/grpc"

	"github.com/go-kit/log"
	"github.com/oklog/oklog/pkg/group"
)

const (
	defaultCORSOrigin    = "http://127.0.0.1:5500"
	defaultHTTPPort      = "12001"
	defaultGRPCPort      = "9090"
	defaultRedisHost     = "redis"
	defaultRedisPort     = "6379"
	defaultRedisDatabase = "0"
	defaultMongoHost     = "mongo"
	defaultMongoPort     = "27017"
)

func main() {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "SERVICE", "GOVERNANCE")

	var (
		// httpAddr = net.JoinHostPort("localhost", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		httpAddr       = net.JoinHostPort("0.0.0.0", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		grpcAddr       = net.JoinHostPort("0.0.0.0", envString("GOVERNANCE_GRPC_PORT", defaultGRPCPort))
		https          = envString("HTTPS", "false")
		secretKey      = requireEnv("SECRET_KEY", logger)
		redisHost      = envString("REDIS_HOST", defaultRedisHost)
		redisPort      = envString("REDIS_PORT", defaultRedisPort)
		reddisPassword = requireEnv("REDIS_PASSWORD", logger)
		redisDatabase  = envString("REDIS_DATABASE_GOVERNANCE", defaultRedisDatabase)
		mongoHost      = envString("MONGO_HOST", defaultMongoHost)
		mongoPort      = envString("MONGO_PORT", defaultMongoPort)
		mongoUsername  = requireEnv("MONGO_INITDB_ROOT_USERNAME", logger)
		mongoPassword  = requireEnv("MONGO_INITDB_ROOT_PASSWORD", logger)
		corsOrigin     = envString("CORS_ORIGIN", defaultCORSOrigin)
	)

	requireFile("config/auth_model.conf", logger)
	if https == "true" {
		requireFile("config/cert.pem", logger)
		requireFile("config/key.pem", logger)
	}

	redisDB, err := strconv.Atoi(redisDatabase)
	if err != nil || redisDB < 0 || redisDB > 15 {
		(&LogEntry{
			Level:   "FATAL",
			Message: "error starting governance",
			Error:   "REDIS_DATABASE_GOVERNANCE must be an integer between 0 and 15",
		}).Log(logger)
		os.Exit(1)
	}

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s/", mongoUsername, mongoPassword, mongoHost, mongoPort)
	mongoService, err := persist.NewMongoService(mongoURI)
	if err != nil {
		(&LogEntry{
			Level:   "FATAL",
			Message: "error starting governance",
			Error:   fmt.Sprintf("failed to create MongoService: %s", err),
		}).Log(logger)
		os.Exit(1)
	}

	casbinService, err := auth.NewService(mongoURI)
	if err != nil {
		(&LogEntry{
			Level:   "FATAL",
			Message: "error starting governance",
			Error:   fmt.Sprintf("failed to create CasbinService: %s", err),
		}).Log(logger)
		os.Exit(1)
	}

	var (
		corsConfig = &transport.CorsOptions{
			Origin:           corsOrigin,                                                                                                  // Allow only specified origin
			Methods:          []string{"GET", "POST", "OPTIONS", "PUT", "UPDATE", "DELETE"},                                               // Allow common HTTP methods
			Headers:          []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "X-Refresh-Token"}, // Allow common HTTP headers
			ExposeHeaders:    []string{"Content-Type", "Authorization", "X-Refresh-Token"},                                                // Allow any headers to be exposed
			AllowCredentials: false,                                                                                                       // Allow credentials to be included
		}
		redisService = persist.NewRedisService(
			fmt.Sprintf("%s:%s", redisHost, redisPort),
			reddisPassword, redisDB,
		)
		service     = governance.NewService(secretKey, redisService, mongoService, casbinService)
		eps         = endpoints.NewEndpointSet(service)
		httpHandler = transport.NewHTTPHandler(eps, secretKey, corsConfig, redisService, casbinService)
		grpcServer  = transport.NewGRPCServer(eps)
	)

	var g group.Group

	{
		// http
		srv := &http.Server{
			Addr:    httpAddr,
			Handler: httpHandler,
		}
		var actor string = "HTTP"
		if https == "true" {
			actor = "HTTPS"
		}
		g.Add(func() error {
			(&LogEntry{
				Level:   "INFO",
				Actor:   actor,
				Message: fmt.Sprintf("starting server on %s", httpAddr),
			}).Log(logger)
			if https == "true" {
				return srv.ListenAndServeTLS("config/cert.pem", "config/key.pem")
			}
			// http
			return srv.ListenAndServe()
		}, func(err error) {
			(&LogEntry{
				Level:   "INFO",
				Actor:   actor,
				Message: "shutting down server",
				Error:   err.Error(),
			}).Log(logger)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
	}

	{
		// grpc
		grpcListener, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			(&LogEntry{
				Level:   "FATAL",
				Message: "error starting governance",
				Error:   fmt.Sprintf("failed to create gRPC listener: %s", err),
			}).Log(logger)
			os.Exit(1)
		}
		baseServer := grpc.NewServer()
		pb.RegisterGovernanceServer(baseServer, grpcServer)
		g.Add(func() error {
			(&LogEntry{
				Level:   "INFO",
				Actor:   "GRPC",
				Message: fmt.Sprintf("starting server on %s", grpcAddr),
			}).Log(logger)
			return baseServer.Serve(grpcListener)
		}, func(err error) {
			(&LogEntry{
				Level:   "INFO",
				Actor:   "GRPC",
				Message: "shutting down server",
				Error:   err.Error(),
			}).Log(logger)
			baseServer.Stop()
		})
	}

	{
		// redis
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_, err := redisService.Ping(ctx)
					if err != nil {
						(&LogEntry{
							Level:   "ERROR",
							Actor:   "RedisService",
							Message: "status disconnected",
							Error:   err.Error(),
						}).Log(logger)
						return err
					} else {
						(&LogEntry{
							Level:   "INFO",
							Actor:   "RedisService",
							Message: "status healthy",
						}).Log(logger)
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(err error) {
			cancel()
			(&LogEntry{
				Level:   "INFO",
				Actor:   "RedisService",
				Message: "closing redis client",
				Error:   err.Error(),
			}).Log(logger)
			_ = redisService.Close()
		})
	}

	{
		// mongo
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					err := mongoService.Ping(ctx)
					if err != nil {
						(&LogEntry{
							Level:   "ERROR",
							Actor:   "MongoService",
							Message: "status disconnected",
							Error:   err.Error(),
						}).Log(logger)
						return err
					} else {
						(&LogEntry{
							Level:   "INFO",
							Actor:   "MongoService",
							Message: "status healthy",
						}).Log(logger)
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(err error) {
			cancel()
			(&LogEntry{
				Level:   "INFO",
				Actor:   "MongoService",
				Message: "disconnecting mongo client",
				Error:   err.Error(),
			}).Log(logger)
			ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelTimeout()
			_ = mongoService.Disconnect(ctx)
		})
	}

	{
		// casbin
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					err := casbinService.LoadPolicy()
					if err != nil {
						(&LogEntry{
							Level:   "ERROR",
							Actor:   "CasbinService",
							Message: "could not reload policy",
							Error:   err.Error(),
						}).Log(logger)
						return err
					} else {
						(&LogEntry{
							Level:   "INFO",
							Actor:   "CasbinService",
							Message: "policy reloaded",
						}).Log(logger)
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(err error) {
			cancel()
			(&LogEntry{
				Level:   "INFO",
				Actor:   "CasbinService",
				Message: "stopping policy reload",
				Error:   err.Error(),
			}).Log(logger)
		})
	}

	{
		// handles Ctrl+C
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return (&LogEntry{
					Level:   "INFO",
					Actor:   "Ctrl+C",
					Message: fmt.Sprintf("received signal: %s", sig),
				}).Log(logger)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}

	(&LogEntry{
		Level:   "FATAL",
		Message: "stopping governance",
		Error:   g.Run().Error(),
	}).Log(logger)
	os.Exit(1)
}

func envString(env, fallback string) string {
	e, ok := os.LookupEnv(env)
	if e == "" || !ok {
		return fallback
	}
	return e
}

func requireEnv(env string, logger log.Logger) string {
	e, ok := os.LookupEnv(env)
	if e == "" || !ok {
		(&LogEntry{
			Level:   "FATAL",
			Message: "error starting governance",
			Error:   fmt.Sprintf("%s is not set", env),
		}).Log(logger)
		os.Exit(1)
	}
	return e
}

func requireFile(path string, logger log.Logger) {
	_, err := os.Stat(path)
	if err != nil {
		(&LogEntry{
			Level:   "FATAL",
			Message: "error starting governance",
			Error:   err.Error(),
		}).Log(logger)
		os.Exit(1)
	}
}

type LogEntry struct {
	Level   string `key:"LEVEL"`
	Actor   string `key:"ACTOR"`
	Message string `key:"MESSAGE"`
	Error   string `key:"ERROR"`
}

func (e *LogEntry) Log(logger log.Logger) error {
	v := reflect.ValueOf(*e)
	t := v.Type()

	var keyvals []any
	for i, fieldType := range reflect.VisibleFields(t) {
		fieldVal := v.Field(i)
		tag := fieldType.Tag.Get("key")

		switch fieldVal.Kind() {
		case reflect.String:
			if val := fieldVal.String(); val != "" {
				keyvals = append(keyvals, tag, val)
			}
		case reflect.Int:
			if val := fieldVal.Int(); val != 0 {
				keyvals = append(keyvals, tag, val)
			}
		}
	}

	if len(keyvals) > 0 {
		return logger.Log(keyvals...)
	}
	return nil
}

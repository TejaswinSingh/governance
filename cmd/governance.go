package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/Resolyte-official/service-governance/pkg/pb"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	"github.com/Resolyte-official/service-governance/pkg/transport"
	"google.golang.org/grpc"

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
	lw := logging.NewTextLogWriter(os.Stdout, logging.INFO)
	mainLogger := logging.NewContextLogger(lw, "Main")

	var (
		// httpAddr = net.JoinHostPort("localhost", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		httpAddr       = net.JoinHostPort("0.0.0.0", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		grpcAddr       = net.JoinHostPort("0.0.0.0", envString("GOVERNANCE_GRPC_PORT", defaultGRPCPort))
		https          = envString("HTTPS", "false")
		secretKey      = requireEnv("SECRET_KEY", mainLogger)
		redisHost      = envString("REDIS_HOST", defaultRedisHost)
		redisPort      = envString("REDIS_PORT", defaultRedisPort)
		reddisPassword = requireEnv("REDIS_PASSWORD", mainLogger)
		redisDatabase  = envString("REDIS_DATABASE_GOVERNANCE", defaultRedisDatabase)
		mongoHost      = envString("MONGO_HOST", defaultMongoHost)
		mongoPort      = envString("MONGO_PORT", defaultMongoPort)
		mongoUsername  = requireEnv("MONGO_INITDB_ROOT_USERNAME", mainLogger)
		mongoPassword  = requireEnv("MONGO_INITDB_ROOT_PASSWORD", mainLogger)
		corsOrigin     = envString("CORS_ORIGIN", defaultCORSOrigin)
	)

	requireFile("config/auth_model.conf", mainLogger)
	if https == "true" {
		requireFile("config/cert.pem", mainLogger)
		requireFile("config/key.pem", mainLogger)
	}

	redisDB, err := strconv.Atoi(redisDatabase)
	if err != nil || redisDB < 0 || redisDB > 15 {
		mainLogger.Fatal(errors.New("REDIS_DATABASE_GOVERNANCE must be an integer between 0 and 15"), "error starting governance")
		os.Exit(1)
	}

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s/", mongoUsername, mongoPassword, mongoHost, mongoPort)
	mongoService, err := persist.NewMongoService(mongoURI)
	if err != nil {
		mainLogger.Fatal(fmt.Errorf("failed to create MongoService: %s", err), "error starting governance")
		os.Exit(1)
	}

	casbinService, err := auth.NewService(mongoURI)
	if err != nil {
		mainLogger.Fatal(fmt.Errorf("failed to create CasbinService: %s", err), "error starting governance")
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
		var mode string = "HTTP"
		if https == "true" {
			mode = "HTTPS"
		}
		g.Add(func() error {
			mainLogger.Infof("starting %s server on %s", mode, httpAddr)
			if https == "true" {
				return srv.ListenAndServeTLS("config/cert.pem", "config/key.pem")
			}
			// http
			return srv.ListenAndServe()
		}, func(err error) {
			mainLogger.ErrorNtf(err, "shutting down %s server", mode)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
	}

	{
		// grpc
		grpcListener, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			mainLogger.Fatal(fmt.Errorf("failed to create gRPC listener: %s", err), "error starting governance")
			os.Exit(1)
		}
		baseServer := grpc.NewServer()
		pb.RegisterGovernanceServer(baseServer, grpcServer)
		g.Add(func() error {
			mainLogger.Infof("starting GRPC server on %s", grpcAddr)
			return baseServer.Serve(grpcListener)
		}, func(err error) {
			mainLogger.ErrorNt(err, "shutting down GRPC server")
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
						mainLogger.Error(err, "RedisService | status disconnected")
						return err
					} else {
						mainLogger.Info("RedisService | status healthy")
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(err error) {
			cancel()
			mainLogger.ErrorNt(err, "RedisService | closing redis client")
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
						mainLogger.Error(err, "MongoService | status disconnected")
						return err
					} else {
						mainLogger.Info("MongoService | status healthy")
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(err error) {
			cancel()
			mainLogger.ErrorNt(err, "MongoService | disconnecting mongo  client")
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
						mainLogger.Error(err, "CasbinService | could not reload policy")
						return err
					} else {
						mainLogger.Info("CasbinService | policy reloaded")
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(err error) {
			cancel()
			mainLogger.ErrorNt(err, "CasbinService | stopping policy reload")
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
				mainLogger.Infof("received signal: %s", sig)
				return nil
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}

	mainLogger.Fatal(g.Run(), "stopping governance")
	os.Exit(1)
}

func envString(env, fallback string) string {
	e, ok := os.LookupEnv(env)
	if e == "" || !ok {
		return fallback
	}
	return e
}

func requireEnv(env string, logger logging.Logger) string {
	e, ok := os.LookupEnv(env)
	if e == "" || !ok {
		logger.Fatal(fmt.Errorf("%s is not set", env), "error starting governance")
		os.Exit(1)
	}
	return e
}

func requireFile(path string, logger logging.Logger) {
	_, err := os.Stat(path)
	if err != nil {
		logger.Fatal(err, "error starting governance")
		os.Exit(1)
	}
}

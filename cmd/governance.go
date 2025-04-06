package main

import (
	"context"
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
	"github.com/Resolyte-official/service-governance/pkg/redis"
	"github.com/Resolyte-official/service-governance/pkg/transport"

	"github.com/go-kit/log"
	"github.com/oklog/oklog/pkg/group"
)

const (
	defaultHTTPPort      = "12001"
	defaultRedisHost     = "redis"
	defaultRedisPort     = "6379"
	defaultRedisDatabase = "0"
	defaultCORSOrigin    = "http://127.0.0.1:5500"
)

func main() {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	var (
		// httpAddr = net.JoinHostPort("localhost", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		httpAddr       = net.JoinHostPort("0.0.0.0", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		https          = envString("HTTPS", "false")
		secretKey      = requireEnv("SECRET_KEY", logger)
		redisHost      = envString("REDIS_HOST", defaultRedisHost)
		redisPort      = envString("REDIS_PORT", defaultRedisPort)
		reddisPassword = requireEnv("REDIS_PASSWORD", logger)
		redisDatabase  = envString("REDIS_DATABASE_GOVERNANCE", defaultRedisDatabase)
		corsOrigin     = envString("CORS_ORIGIN", defaultCORSOrigin)
	)

	if https == "true" {
		requireFile("config/cert.pem", logger)
		requireFile("config/key.pem", logger)
	}

	redisDB, err := strconv.Atoi(redisDatabase)
	if err != nil || redisDB < 0 || redisDB > 15 {
		logger.Log("exit", "REDIS_DATABASE_GOVERNANCE must be an integer between 0 and 15")
		os.Exit(1)
	}

	casbinService, err := auth.NewService()
	if err != nil {
		logger.Log("exit", "Failed to create casbin service", "error", err)
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
		redisService = redis.NewService(
			fmt.Sprintf("%s:%s", redisHost, redisPort),
			reddisPassword, redisDB,
		)
		service     = governance.NewService(secretKey, redisService)
		eps         = endpoints.NewEndpointSet(service)
		httpHandler = transport.NewHTTPHandler(eps, secretKey, corsConfig, redisService, casbinService)
	)

	var g group.Group

	{
		srv := &http.Server{
			Addr:    httpAddr,
			Handler: httpHandler,
		}
		g.Add(func() error {
			if https == "true" {
				logger.Log("transport", "HTTPS", "addr", httpAddr)
				return srv.ListenAndServeTLS("config/cert.pem", "config/key.pem")
			}
			// http
			logger.Log("transport", "HTTP", "addr", httpAddr)
			return srv.ListenAndServe()
		}, func(err error) {
			logger.Log("exit", "shutting down HTTP server", "error", err)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
	}

	{
		// redis
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_, err := redisService.Ping(ctx)
					if err != nil {
						logger.Log("service", "Redis", "status", "disconnected", "error", err)
						return nil
					} else {
						logger.Log("service", "Redis", "status", "healthy")
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(error) {
			cancel()
			redisService.Close()
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
				return fmt.Errorf("received signal %s", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}
	logger.Log("exit", g.Run())
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
		logger.Log("exit", fmt.Sprintf("%s is not set", env))
		os.Exit(1)
	}
	return e
}

func requireFile(path string, logger log.Logger) {
	_, err := os.Stat(path)
	if err != nil {
		logger.Log("exit", err)
		os.Exit(1)
	}
}

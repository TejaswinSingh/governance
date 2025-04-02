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
	defaultHTTPPort      = "12000"
	defaultRedisHost     = "redis"
	defaultRedisPort     = "6379"
	defaultRedisDatabase = "0"
)

func main() {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	var (
		// httpAddr = net.JoinHostPort("localhost", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		httpAddr       = net.JoinHostPort("0.0.0.0", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		secretKey      = requireEnv("SECRET_KEY", logger)
		redisHost      = envString("REDIS_HOST", defaultRedisHost)
		redisPort      = envString("REDIS_PORT", defaultRedisPort)
		reddisPassword = requireEnv("REDIS_PASSWORD", logger)
		redisDatabase  = envString("REDIS_DATABASE_GOVERNANCE", defaultRedisDatabase)
	)

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
		RedisService = redis.NewService(
			fmt.Sprintf("%s:%s", redisHost, redisPort),
			reddisPassword, redisDB,
		)
		service     = governance.NewService(secretKey, RedisService)
		eps         = endpoints.NewEndpointSet(service)
		httpHandler = transport.NewHTTPHandler(eps, secretKey, RedisService, casbinService)
	)

	var g group.Group

	{
		httpListener, err := net.Listen("tcp", httpAddr)
		if err != nil {
			logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "HTTP", "addr", httpAddr)
			return http.Serve(httpListener, httpHandler)
		}, func(error) {
			httpListener.Close()
		})
	}

	{
		// redis
		ctx := context.Background()
		g.Add(func() error {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_, err := RedisService.Ping(ctx)
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
			RedisService.Close()
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

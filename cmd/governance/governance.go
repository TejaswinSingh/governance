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

	"github.com/TejaswinSingh/version0/services/governance"
	"github.com/TejaswinSingh/version0/services/governance/endpoints"
	"github.com/TejaswinSingh/version0/services/governance/redis"
	"github.com/TejaswinSingh/version0/services/governance/transport"
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
	var (
		logger log.Logger
		// httpAddr = net.JoinHostPort("localhost", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		httpAddr       = net.JoinHostPort("0.0.0.0", envString("GOVERNANCE_HTTP_PORT", defaultHTTPPort))
		secretKey      = envString("SECRET_KEY", "")
		redisHost      = envString("REDIS_HOST", defaultRedisHost)
		redisPort      = envString("REDIS_PORT", defaultRedisPort)
		reddisPassword = envString("REDIS_PASSWORD", "")
		redisDatabase  = envString("REDIS_DATABASE_GOVERNANCE", defaultRedisDatabase)
	)

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	if secretKey == "" {
		logger.Log("exit", "SECRET_KEY is not set")
		os.Exit(1)
	}
	if redisHost == "" {
		logger.Log("exit", "REDIS_HOST is not set")
		os.Exit(1)
	}
	if redisPort == "" {
		logger.Log("exit", "REDIS_PORT is not set")
		os.Exit(1)
	}
	if reddisPassword == "" {
		logger.Log("exit", "REDIS_PASSWORD is not set")
		os.Exit(1)
	}
	redisDB, err := strconv.Atoi(redisDatabase)
	if err != nil {
		logger.Log("exit", "REDIS_DATABASE_GOVERNANCE is not set correctly")
		os.Exit(1)
	}

	var (
		RedisService = redis.NewService(
			fmt.Sprintf("%s:%s", redisHost, redisPort),
			reddisPassword, redisDB,
		)
		service     = governance.NewService(secretKey, RedisService)
		eps         = endpoints.NewEndpointSet(service)
		httpHandler = transport.NewHTTPHandler(eps, secretKey, RedisService)
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
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

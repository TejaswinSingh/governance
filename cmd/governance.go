package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	governance "github.com/Resolyte-official/service-governance/pkg"
	"github.com/Resolyte-official/service-governance/pkg/auth"
	"github.com/Resolyte-official/service-governance/pkg/config"
	"github.com/Resolyte-official/service-governance/pkg/endpoints"
	"github.com/Resolyte-official/service-governance/pkg/logging"
	"github.com/Resolyte-official/service-governance/pkg/pb"
	"github.com/Resolyte-official/service-governance/pkg/persist"
	"github.com/Resolyte-official/service-governance/pkg/transport"
	"google.golang.org/grpc"

	"github.com/oklog/oklog/pkg/group"
)

func main() {
	envConfig := config.NewEnvironmentConfiguration()

	var lw logging.LogWriter
	if envConfig.Env.IsProduction() {
		lw = logging.NewJsonLogWriter(os.Stdout, logging.INFO)
	} else {
		lw = logging.NewTextLogWriter(os.Stdout, logging.INFO)
	}

	mainLogger := logging.NewContextLogger(lw, "Main")

	mainLogger.Infof("starting governance in %s mode", envConfig.Env)

	requireFile("config/auth_model.conf")
	if envConfig.HttpMode.IsHttps() { // need certs
		requireFile("config/cert.pem")
		requireFile("config/key.pem")
	}

	mongoURI := persist.CreateMongoURI(&envConfig.Mongo)
	mongoService, err := persist.NewMongoService(mongoURI)
	if err != nil {
		mainLogger.Fatal(fmt.Errorf("failed to create MongoService: %s", err), "error starting governance")
	}

	casbinService, err := auth.NewService(mongoURI)
	if err != nil {
		mainLogger.Fatal(fmt.Errorf("failed to create CasbinService: %s", err), "error starting governance")
	}

	var (
		corsConfig = &transport.CorsOptions{
			Origin:           envConfig.CorsOrigin,                                                                                        // Allow only specified origin
			Methods:          []string{"GET", "POST", "OPTIONS", "PUT", "UPDATE", "DELETE"},                                               // Allow common HTTP methods
			Headers:          []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "X-Refresh-Token"}, // Allow common HTTP headers
			ExposeHeaders:    []string{"Content-Type", "Authorization", "X-Refresh-Token"},                                                // Allow any headers to be exposed
			AllowCredentials: false,                                                                                                       // Allow credentials to be included
		}
		redisService = persist.NewRedisService(&envConfig.Redis)
		store        = &governance.ServiceStore{
			Redis:  redisService,
			Mongo:  mongoService,
			Casbin: casbinService,
		}
		service     = governance.NewService(&envConfig, store)
		eps         = endpoints.NewEndpointSet(service)
		httpHandler = transport.NewHTTPHandler(eps, &envConfig, corsConfig, store, lw)
		grpcServer  = transport.NewGRPCServer(eps, lw)
	)

	var g group.Group

	{
		// http
		httpAddr := net.JoinHostPort("0.0.0.0", envConfig.HttpPort) // net.JoinHostPort("localhost", envConfig.HttpPort)
		srv := &http.Server{
			Addr:    httpAddr,
			Handler: httpHandler,
		}
		g.Add(func() error {
			mainLogger.Infof("starting %s server on %s", envConfig.HttpMode, httpAddr)
			if envConfig.HttpMode.IsHttps() {
				return srv.ListenAndServeTLS("config/cert.pem", "config/key.pem")
			}
			// http
			return srv.ListenAndServe()
		}, func(err error) {
			mainLogger.ErrorNtf(err, "shutting down %s server", envConfig.HttpMode)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
	}

	{
		// grpc
		grpcAddr := net.JoinHostPort("0.0.0.0", envConfig.GrpcPort)
		grpcListener, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			mainLogger.Fatal(fmt.Errorf("failed to create gRPC listener: %s", err), "error starting governance")
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
}

func requireFile(path string) {
	_, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
}

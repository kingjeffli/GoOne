package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Iori372552686/GoOne/common/gamedata"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/service/bootstrap"
	"github.com/Iori372552686/GoOne/lib/util/sensitive_words"
	"github.com/Iori372552686/GoOne/lib/web/web_gin"
	"github.com/Iori372552686/GoOne/module/misc"
	"github.com/Iori372552686/GoOne/src/web_svr/controller"
	"github.com/Iori372552686/GoOne/src/web_svr/globals"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type webRuntime struct {
	httpSrv *http.Server
	grpcSrv *grpc.Server
}

func newApp() *bootstrap.ServiceApp {
	runtime := &webRuntime{}
	return bootstrap.NewServiceApp(bootstrap.Options{
		ServiceName: "websvr",
		LoadConfig: func() error {
			if err := gconf.LoadWebConfig(*gconf.SvrConfFile); err != nil {
				return err
			}
			logger.Infof("svr_conf: %+v", gconf.WebSvrCfg)
			if gconf.WebSvrCfg.Dependencies.GameDataDir != "" {
				logger.Infof("Loading local file by gameconf_dir: %v ", gconf.WebSvrCfg.Dependencies.GameDataDir)
				if err := gamedata.InitLocal(gconf.WebSvrCfg.Dependencies.GameDataDir); err != nil {
					return err
				}
			}
			return nil
		},
		LoggerConfig: func() bootstrap.LoggerConfig {
			return bootstrap.LoggerConfig{
				Dir:   gconf.WebSvrCfg.Debug.LogDir,
				Level: gconf.WebSvrCfg.Debug.LogLevel,
				Name:  "websvr",
			}
		},
		AdminConfig: func() bootstrap.AdminConfig {
			return bootstrap.NewAdminConfig(
				"websvr",
				misc.ServerType_WebSvr,
				gconf.WebSvrCfg.CommonRuntime.AdminServer.Enabled,
				gconf.WebSvrCfg.CommonDebug.Pprof,
				gconf.WebSvrCfg.CommonRuntime.AdminServer.IP,
				gconf.WebSvrCfg.CommonRuntime.AdminServer.Port,
			)
		},
		InitDeps: func() error {
			if err := globals.RedisMgr.InitAndRun(gconf.WebSvrCfg.Dependencies.DbInstances); err != nil {
				return err
			}
			globals.SignMgr.InitAndRun(gconf.WebSvrCfg.Dependencies.HTTPSigns)
			globals.RestMgr.Init(gconf.WebSvrCfg.Dependencies.RestApiConf, globals.SignMgr)
			sensitive_words.Init(gconf.WebSvrCfg.Dependencies.SensitiveWordsFile)
			return nil
		},
		StartRuntime: func() error {
			httpSrv, err := web_gin.StartGin(gconf.WebSvrCfg.Runtime.HttpServer, controller.LoadWebRoutes)
			if err != nil {
				return err
			}
			runtime.httpSrv = httpSrv
			if err := runtime.startGRPCServer(); err != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = runtime.shutdown(ctx)
				return err
			}
			return nil
		},
		OnProc: func() bool {
			return true
		},
		OnShutdown: func(ctx context.Context) error {
			return runtime.shutdown(ctx)
		},
		OnExit: func() {
			logger.Infof("================== websvr Stop =========================")
		},
	})
}

func (r *webRuntime) startGRPCServer() error {
	conf := gconf.WebSvrCfg.Runtime.GRPCServer
	if !conf.Enabled {
		return nil
	}
	if conf.Port <= 0 {
		return errors.New("grpc_server.port args err!")
	}

	addr := fmt.Sprintf("%s:%d", conf.IP, conf.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	d, _ := controller.BuildWebDispatcher()
	srv := grpc.NewServer()
	d.MountGRPC(srv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("web.websvr.v1.WebApiService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	reflection.Register(srv)
	r.grpcSrv = srv

	go func() {
		logger.Infof("------ gRPC Server Running by %v ------", addr)
		if err := srv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Errorf("gRPC Serve error !! err=%v", err)
		}
	}()
	return nil
}

func (r *webRuntime) shutdown(ctx context.Context) error {
	var shutdownErr error
	if r.grpcSrv != nil {
		done := make(chan struct{})
		grpcSrv := r.grpcSrv
		go func() {
			grpcSrv.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			grpcSrv.Stop()
			if shutdownErr == nil {
				shutdownErr = ctx.Err()
			}
		}
		r.grpcSrv = nil
	}
	if r.httpSrv != nil {
		if err := r.httpSrv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("http shutdown error | %v", err)
			if shutdownErr == nil {
				shutdownErr = err
			}
		}
		r.httpSrv = nil
	}
	return shutdownErr
}

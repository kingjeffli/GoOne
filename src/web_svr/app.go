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
	"github.com/Iori372552686/GoOne/lib/util/marshal"
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
			if err := marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.WebSvrCfg); err != nil {
				return err
			}
			logger.Infof("svr_conf: %+v", gconf.WebSvrCfg)
			if gconf.WebSvrCfg.GameDataDir != "" {
				logger.Infof("Loading local file by gameconf_dir: %v ", gconf.WebSvrCfg.GameDataDir)
				if err := gamedata.InitLocal(gconf.WebSvrCfg.GameDataDir); err != nil {
					return err
				}
			}
			return nil
		},
		LoggerConfig: func() bootstrap.LoggerConfig {
			return bootstrap.LoggerConfig{
				Dir:   gconf.WebSvrCfg.LogDir,
				Level: gconf.WebSvrCfg.LogLevel,
				Name:  "websvr",
			}
		},
		AdminConfig: func() bootstrap.AdminConfig {
			return bootstrap.NewAdminConfig(
				"websvr",
				misc.ServerType_WebSvr,
				gconf.WebSvrCfg.AdminServer.Enabled,
				gconf.WebSvrCfg.Pprof,
				gconf.WebSvrCfg.AdminServer.IP,
				gconf.WebSvrCfg.AdminServer.Port,
			)
		},
		InitDeps: func() error {
			if err := globals.RedisMgr.InitAndRun(gconf.WebSvrCfg.DbInstances); err != nil {
				return err
			}
			globals.SignMgr.InitAndRun(gconf.WebSvrCfg.HTTPSigns)
			globals.RestMgr.Init(gconf.WebSvrCfg.RestApiConf, globals.SignMgr)
			sensitive_words.Init(gconf.WebSvrCfg.SensitiveWordsFile)
			return nil
		},
		StartRuntime: func() error {
			httpSrv, err := web_gin.StartGin(gconf.WebSvrCfg.HttpServer, controller.LoadWebRoutes)
			if err != nil {
				return err
			}
			runtime.httpSrv = httpSrv
			if err := runtime.startGRPCServer(); err != nil {
				runtime.shutdown()
				return err
			}
			return nil
		},
		OnProc: func() bool {
			return true
		},
		OnExit: func() {
			runtime.shutdown()
			logger.Infof("================== websvr Stop =========================")
		},
	})
}

func (r *webRuntime) startGRPCServer() error {
	conf := gconf.WebSvrCfg.GRPCServer
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

func (r *webRuntime) shutdown() {
	if r.grpcSrv != nil {
		r.grpcSrv.GracefulStop()
		r.grpcSrv = nil
	}
	if r.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := r.httpSrv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("http shutdown error | %v", err)
		}
		r.httpSrv = nil
	}
}

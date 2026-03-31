package main

import (
	connsvrv1 "github.com/Iori372552686/GoOne/api/gen/game/connsvr/v1"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/service/bootstrap"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/lib/util/marshal"
	"github.com/Iori372552686/GoOne/module/misc"
	"github.com/Iori372552686/GoOne/src/connsvr/globals"
	"github.com/Iori372552686/GoOne/src/connsvr/service"
)

func newApp() *bootstrap.ServiceApp {
	return bootstrap.NewServiceApp(bootstrap.Options{
		ServiceName: "connsvr",
		LoadConfig: func() error {
			if err := marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.ConnSvrCfg); err != nil {
				return err
			}
			logger.Infof("svr_conf: %+v", gconf.ConnSvrCfg)
			return nil
		},
		LoggerConfig: func() bootstrap.LoggerConfig {
			return bootstrap.LoggerConfig{
				Dir:   gconf.ConnSvrCfg.LogDir,
				Level: gconf.ConnSvrCfg.LogLevel,
				Name:  "connsvr",
			}
		},
		AdminConfig: func() bootstrap.AdminConfig {
			return bootstrap.NewAdminConfig(
				"connsvr",
				misc.ServerType_ConnSvr,
				gconf.ConnSvrCfg.AdminServer.Enabled,
				gconf.ConnSvrCfg.Pprof,
				gconf.ConnSvrCfg.AdminServer.IP,
				gconf.ConnSvrCfg.AdminServer.Port,
			)
		},
		InitDeps: func() error {
			globals.SignMgr.InitAndRun(gconf.ConnSvrCfg.HTTPSigns)
			globals.RestMgr.Init(gconf.ConnSvrCfg.RestApiConf, globals.SignMgr)
			return nil
		},
		RegisterHandlers: func() error {
			srv := connsvrv1.NewConnServiceSServer(&service.ConnServiceImpl{}, ssrpc.DefaultMWOptions{})
			d := ssrpc.NewDispatcher()
			connsvrv1.RegisterConnServiceToDispatcher(d, srv)
			d.RegisterToTransactionMgr(globals.TransMgr)
			return nil
		},
		StartRuntime: func() error {
			globals.TransMgr.InitAndRun(misc.MaxTransNumber, false, 0)
			if err := router.InitAndRun(
				gconf.ConnSvrCfg.SelfBusId,
				onRecvSSPacket,
				gconf.ConnSvrCfg.BusMQAddr,
				misc.ServerRouteRules,
				gconf.ConnSvrCfg.RegisterAddr,
			); err != nil {
				return err
			}
			if err := globals.ConnTcpSvr.CreateTcpServer("", gconf.ConnSvrCfg.ListenPort+1, onTcpPacket); err != nil {
				return err
			}
			return globals.ConnWsSvr.CreateWebSocketServer("gin", "debug", gconf.ConnSvrCfg.ListenPort, onWebSocketPacket)
		},
		OnProc: func() bool {
			return true
		},
		OnExit: func() {
			logger.Infof("================== connsvr Stop =========================")
		},
	})
}

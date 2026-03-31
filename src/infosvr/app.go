package main

import (
	infosvrv1 "github.com/Iori372552686/GoOne/api/gen/game/infosvr/v1"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/service/bootstrap"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/lib/util/marshal"
	"github.com/Iori372552686/GoOne/module/misc"
	"github.com/Iori372552686/GoOne/src/infosvr/globals"
	"github.com/Iori372552686/GoOne/src/infosvr/service"
)

func onRecvSSPacket(packet *sharedstruct.SSPacket) {
	globals.TransMgr.ProcessSSPacket(packet)
	packet = nil // packet所有权转交给transmgr，后面不能再用packet（包括data）
}

func newApp() *bootstrap.ServiceApp {
	return bootstrap.NewServiceApp(bootstrap.Options{
		ServiceName: "infosvr",
		LoadConfig: func() error {
			return marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.InfoSvrCfg)
		},
		LoggerConfig: func() bootstrap.LoggerConfig {
			return bootstrap.LoggerConfig{
				Dir:   gconf.InfoSvrCfg.LogDir,
				Level: gconf.InfoSvrCfg.LogLevel,
				Name:  "infosvr",
			}
		},
		AdminConfig: func() bootstrap.AdminConfig {
			return bootstrap.NewAdminConfig(
				"infosvr",
				misc.ServerType_InfoSvr,
				gconf.InfoSvrCfg.AdminServer.Enabled,
				gconf.InfoSvrCfg.Pprof,
				gconf.InfoSvrCfg.AdminServer.IP,
				gconf.InfoSvrCfg.AdminServer.Port,
			)
		},
		InitDeps: func() error {
			return globals.InfoMgr.RedisMgr.InitAndRun(gconf.InfoSvrCfg.DbInstances)
		},
		RegisterHandlers: func() error {
			srv := infosvrv1.NewInfoServiceSServer(&service.InfoServiceImpl{}, ssrpc.DefaultMWOptions{})
			d := ssrpc.NewDispatcher()
			infosvrv1.RegisterInfoServiceToDispatcher(d, srv)
			d.RegisterToTransactionMgr(globals.TransMgr)
			return nil
		},
		StartRuntime: func() error {
			globals.TransMgr.InitAndRun(misc.MaxTransNumber, false, 0)
			return router.InitAndRun(
				gconf.InfoSvrCfg.SelfBusId,
				onRecvSSPacket,
				gconf.InfoSvrCfg.BusMQAddr,
				misc.ServerRouteRules,
				gconf.InfoSvrCfg.RegisterAddr,
			)
		},
		OnProc: func() bool {
			return true
		},
		OnExit: func() {
			logger.Infof("================== infosvr Stop =========================")
		},
	})
}

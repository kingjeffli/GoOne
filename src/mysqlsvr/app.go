package main

import (
	mysqlsvrv1 "github.com/Iori372552686/GoOne/api/gen/game/mysqlsvr/v1"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/service/bootstrap"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/lib/util/marshal"
	"github.com/Iori372552686/GoOne/module/misc"
	"github.com/Iori372552686/GoOne/src/mysqlsvr/globals"
	"github.com/Iori372552686/GoOne/src/mysqlsvr/manager"
	"github.com/Iori372552686/GoOne/src/mysqlsvr/service"
)

func onRecvSSPacket(packet *sharedstruct.SSPacket) {
	globals.TransMgr.ProcessSSPacket(packet)
	packet = nil // packet所有权转交给transmgr，后面不能再用packet（包括data）
}

func newApp() *bootstrap.ServiceApp {
	return bootstrap.NewServiceApp(bootstrap.Options{
		ServiceName: "mysqlsvr",
		LoadConfig: func() error {
			return marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.MySqlSvrCfg)
		},
		LoggerConfig: func() bootstrap.LoggerConfig {
			return bootstrap.LoggerConfig{
				Dir:   gconf.MySqlSvrCfg.LogDir,
				Level: gconf.MySqlSvrCfg.LogLevel,
				Name:  "mysqlsvr",
			}
		},
		AdminConfig: func() bootstrap.AdminConfig {
			return bootstrap.NewAdminConfig(
				"mysqlsvr",
				misc.ServerType_MysqlSvr,
				gconf.MySqlSvrCfg.AdminServer.Enabled,
				gconf.MySqlSvrCfg.Pprof,
				gconf.MySqlSvrCfg.AdminServer.IP,
				gconf.MySqlSvrCfg.AdminServer.Port,
			)
		},
		InitDeps: func() error {
			return globals.OrmMgr.InitAndRun(gconf.MySqlSvrCfg.OrmConf, manager.GetTables()...)
		},
		RegisterHandlers: func() error {
			srv := mysqlsvrv1.NewMysqlServiceSServer(&service.MysqlServiceImpl{}, ssrpc.DefaultMWOptions{})
			d := ssrpc.NewDispatcher()
			mysqlsvrv1.RegisterMysqlServiceToDispatcher(d, srv)
			d.RegisterToTransactionMgr(globals.TransMgr)
			return nil
		},
		StartRuntime: func() error {
			globals.TransMgr.InitAndRun(misc.MaxTransNumber, false, 0)
			return router.InitAndRun(
				gconf.MySqlSvrCfg.SelfBusId,
				onRecvSSPacket,
				gconf.MySqlSvrCfg.BusMQAddr,
				misc.ServerRouteRules,
				gconf.MySqlSvrCfg.RegisterAddr,
			)
		},
		OnProc: func() bool {
			return true
		},
		OnExit: func() {
			manager.Close()
			globals.MysqlMgr.Destroy()
			logger.Infof("================== mysqlsvr Stop =========================")
		},
	})
}

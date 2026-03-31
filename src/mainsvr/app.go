package main

import (
	mainsvrv1 "github.com/Iori372552686/GoOne/api/gen/game/mainsvr/v1"
	"github.com/Iori372552686/GoOne/common/gamedata"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/datetime"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/net_conf"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/service/bootstrap"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/lib/service/transaction"
	"github.com/Iori372552686/GoOne/lib/util/idgen"
	"github.com/Iori372552686/GoOne/lib/util/marshal"
	"github.com/Iori372552686/GoOne/lib/util/safego"
	"github.com/Iori372552686/GoOne/lib/util/sensitive_words"
	"github.com/Iori372552686/GoOne/module/misc"
	"github.com/Iori372552686/GoOne/src/mainsvr/globals"
	"github.com/Iori372552686/GoOne/src/mainsvr/globals/rds"
	"github.com/Iori372552686/GoOne/src/mainsvr/service"
)

func onRecvSSPacket(packet *sharedstruct.SSPacket) {
	globals.TransMgr.ProcessSSPacket(packet)
	packet = nil // packet所有权转交给transmgr，后面不能再用packet（包括data）
}

func newApp() *bootstrap.ServiceApp {
	return bootstrap.NewServiceApp(bootstrap.Options{
		ServiceName: "mainsvr",
		LoadConfig: func() error {
			if err := marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.MainSvrCfg); err != nil {
				return err
			}
			if gconf.MainSvrCfg.GameDataDir != "" {
				logger.Infof("Loading local file by gameconf_dir: %v ", gconf.MainSvrCfg.GameDataDir)
				if err := gamedata.InitLocal(gconf.MainSvrCfg.GameDataDir); err != nil {
					return err
				}
			}
			logger.Infof("gconf file load success | %+v", gconf.MainSvrCfg)
			return nil
		},
		LoggerConfig: func() bootstrap.LoggerConfig {
			return bootstrap.LoggerConfig{
				Dir:   gconf.MainSvrCfg.LogDir,
				Level: gconf.MainSvrCfg.LogLevel,
				Name:  "mainsvr",
			}
		},
		AdminConfig: func() bootstrap.AdminConfig {
			return bootstrap.NewAdminConfig(
				"mainsvr",
				misc.ServerType_MainSvr,
				gconf.MainSvrCfg.AdminServer.Enabled,
				gconf.MainSvrCfg.Pprof,
				gconf.MainSvrCfg.AdminServer.IP,
				gconf.MainSvrCfg.AdminServer.Port,
			)
		},
		InitDeps: func() error {
			sensitive_words.Init(gconf.MainSvrCfg.SensitiveWordsFile)
			if err := rds.RedisMgr.InitAndRun(gconf.MainSvrCfg.DbInstances); err != nil {
				return err
			}
			idGen, err := idgen.NewIDGen()
			if err != nil {
				return err
			}
			globals.IDGen = idGen
			if gconf.MainSvrCfg.NacosConf.IPAddr != "" {
				logger.Infof("Loading remote gameconf by Nacos group: %v ", gconf.MainSvrCfg.NacosConf.GroupName)
				if err := gamedata.InitNet(net_conf.NewNacosConfigClient(gconf.MainSvrCfg.NacosConf), gconf.MainSvrCfg.NacosConf.GroupName); err != nil {
					return err
				}
			}
			return nil
		},
		RegisterHandlers: func() error {
			srv := mainsvrv1.NewMainC2SServiceSServer(&service.MainC2SServiceImpl{}, ssrpc.DefaultMWOptions{})
			d := ssrpc.NewDispatcher()
			mainsvrv1.RegisterMainC2SServiceToDispatcher(d, srv)
			d.RegisterToTransactionMgr(globals.TransMgr)
			return nil
		},
		StartRuntime: func() error {
			transShardCount := gconf.MainSvrCfg.TransShardCount
			if transShardCount <= 0 {
				transShardCount = transaction.DefaultShardCount()
			}
			globals.TransMgr.InitAndRunWithConfig(transaction.TransactionMgrConfig{
				MaxTrans:         misc.MaxTransNumber,
				ShardCount:       transShardCount,
				MaxPendingPerKey: 100,
			})
			logger.Infof("mainsvr transmgr shards=%d serial_key=routerid_or_uid", transShardCount)
			return router.InitAndRun(
				gconf.MainSvrCfg.SelfBusId,
				onRecvSSPacket,
				gconf.MainSvrCfg.BusMQAddr,
				misc.ServerRouteRules,
				gconf.MainSvrCfg.RegisterAddr,
			)
		},
		OnProc: func() bool {
			return true
		},
		OnTick: func(lastMs, nowMs int64) {
			if lastMs/datetime.MS_PER_MINUTE != nowMs/datetime.MS_PER_MINUTE {
				safego.Go(func() { globals.RoleMgr.Tick() })
			}
		},
		OnExit: func() {
			logger.Infof("================== mainsvr Stop =========================")
		},
	})
}

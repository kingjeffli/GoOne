package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"

	mainsvrv1 "github.com/Iori372552686/GoOne/api/gen/game/mainsvr/v1"
	"github.com/Iori372552686/GoOne/module/misc"

	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/util/safego"

	"github.com/Iori372552686/GoOne/common/gamedata"
	"github.com/Iori372552686/GoOne/lib/api/datetime"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/net_conf"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/lib/service/transaction"
	"github.com/Iori372552686/GoOne/lib/util/idgen"
	"github.com/Iori372552686/GoOne/lib/util/marshal"
	"github.com/Iori372552686/GoOne/lib/util/sensitive_words"
	"github.com/Iori372552686/GoOne/src/mainsvr/globals"
	"github.com/Iori372552686/GoOne/src/mainsvr/globals/rds"
	"github.com/Iori372552686/GoOne/src/mainsvr/service"
)

// gameSvr  struct
type MainSvrImpl struct{}

//---------------------------------- func

func onRecvSSPacket(packet *sharedstruct.SSPacket) {
	globals.TransMgr.ProcessSSPacket(packet)
	packet = nil // packet所有权转交给transmgr，后面不能再用packet（包括data）
}

func (self *MainSvrImpl) OnInit() error {
	//-- set sys args
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)

	//-- load cfg
	err := self.OnReload()
	if err != nil {
		logger.Errorf("Failed to load config | %v", err)
		return err
	}

	// init zap logger
	if _, err = logger.InitLogger(gconf.MainSvrCfg.LogDir, gconf.MainSvrCfg.LogLevel, "mainsvr"); err != nil {
		return err
	}

	if gconf.MainSvrCfg.Pprof {
		go func() {
			logger.Infof("pprof listen on :81%02d", misc.ServerType_MainSvr)
			log.Println(http.ListenAndServe(fmt.Sprintf(":81%02d", misc.ServerType_MainSvr), nil))
		}()
	}

	sensitive_words.Init(gconf.MainSvrCfg.SensitiveWordsFile)
	err = router.InitAndRun(gconf.MainSvrCfg.SelfBusId,
		onRecvSSPacket,
		gconf.MainSvrCfg.BusMQAddr,
		misc.ServerRouteRules,
		gconf.MainSvrCfg.RegisterAddr,
	)
	if err != nil {
		logger.Errorf("Failed to initialize router | %v", err)
		return err
	}

	//-- init redis
	err = rds.RedisMgr.InitAndRun(gconf.MainSvrCfg.DbInstances)
	if err != nil {
		logger.Errorf("RedisMgr InitAndRun error !! err=%v", err)
		return err
	}

	// IDL-driven ssrpc handlers.
	// Phase 2: register into a unified dispatcher, then mount into TransactionMgr.
	srv := mainsvrv1.NewMainC2SServiceSServer(&service.MainC2SServiceImpl{}, ssrpc.DefaultMWOptions{})
	d := ssrpc.NewDispatcher()
	mainsvrv1.RegisterMainC2SServiceToDispatcher(d, srv)
	d.RegisterToTransactionMgr(globals.TransMgr)
	transShardCount := gconf.MainSvrCfg.TransShardCount
	if transShardCount <= 0 {
		transShardCount = transaction.DefaultShardCount()
	}
	globals.TransMgr.InitAndRunWithConfig(transaction.TransactionMgrConfig{
		MaxTrans:         misc.MaxTransNumber,
		ShardCount:       transShardCount,
		SerialKeyMode:    transaction.SerialKeyModeUID,
		MaxPendingPerKey: 100,
	})
	logger.Infof("mainsvr transmgr shards=%d serial_key=%s", transShardCount, transaction.SerialKeyModeUID.String())
	if globals.IDGen, err = idgen.NewIDGen(); err != nil {
		return err
	}

	//remote loading gameconf
	if gconf.MainSvrCfg.NacosConf.IPAddr != "" {
		logger.Infof("Loading remote gameconf by Nacos group: %v ", gconf.MainSvrCfg.NacosConf.GroupName)
		err = gamedata.InitNet(net_conf.NewNacosConfigClient(gconf.MainSvrCfg.NacosConf), gconf.MainSvrCfg.NacosConf.GroupName)
		if err != nil {
			return err
		}
	}

	logger.Infof("mainsvr init success")
	return nil
}

func (self *MainSvrImpl) OnReload() error {
	err := marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.MainSvrCfg)
	if err != nil {
		logger.Errorf("Failed to load server config | %v", err)
		return err
	}

	//local loading gameconf
	if gconf.MainSvrCfg.GameDataDir != "" {
		logger.Infof("Loading local file by gameconf_dir: %v ", gconf.MainSvrCfg.GameDataDir)
		err = gamedata.InitLocal(gconf.MainSvrCfg.GameDataDir)
		if err != nil {
			return err
		}
	}

	logger.Infof(" gconf file load success | %+v", gconf.MainSvrCfg)
	return nil
}

/**
* @Description:  proc
* @return: bool
* @Author: Iori
* @Date: 2022-04-27 21:05:01
**/
func (self *MainSvrImpl) OnProc() bool {
	// mainloop  proc
	return true
}

/**
* @Description: mainloop tick
* @param: lastMs
* @param: nowMs
* @Author: Iori
* @Date: 2022-04-27 21:04:53
**/
func (self *MainSvrImpl) OnTick(lastMs, nowMs int64) {
	if lastMs/datetime.MS_PER_MINUTE != nowMs/datetime.MS_PER_MINUTE { // 一分钟调用
		safego.Go(func() { globals.RoleMgr.Tick() })
	}
}

/**
* @Description: main exit
* @Author: Iori
* @Date: 2022-04-27 21:05:07
**/
func (self *MainSvrImpl) OnExit() {
	// game exit todo something
	logger.Flush()
	logger.Infof("service exit, right now !")
	logger.Infof("================== MainSvrImpl Stop =========================")
}

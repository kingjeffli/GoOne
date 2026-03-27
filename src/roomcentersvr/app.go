package main

import (
	roomcenterv1 "github.com/Iori372552686/GoOne/api/gen/game/roomcenter/v1"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/module/misc"
	id "github.com/Iori372552686/GoOne/src/roomcentersvr/globals/idgen"
	"github.com/Iori372552686/GoOne/src/roomcentersvr/service"
	pb "github.com/Iori372552686/game_protocol/protocol"
	"runtime"

	"github.com/Iori372552686/GoOne/common/gamedata"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/net_conf"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/transaction"
	"github.com/Iori372552686/GoOne/lib/util/idgen"
	"github.com/Iori372552686/GoOne/lib/util/marshal"
	"github.com/Iori372552686/GoOne/lib/util/safego"
	"github.com/Iori372552686/GoOne/src/roomcentersvr/globals"
	"github.com/Iori372552686/GoOne/src/roomcentersvr/room_ai"
)

func onRecvSSPacket(packet *sharedstruct.SSPacket) {
	globals.TransMgr.ProcessSSPacket(packet)
	packet = nil // packet所有权转交给transmgr，后面不能再用packet（包括data）
}

type RoomMgrSvrImpl struct {
}

func (a *RoomMgrSvrImpl) OnInit() error {
	//-- set sys args
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)

	//-- load cfg
	err := a.OnReload()
	if err != nil {
		logger.Errorf("Failed to load config | %v", err)
		return err
	}

	// init zap logger
	if _, err = logger.InitLogger(gconf.RoomCenterSvrCfg.LogDir, gconf.RoomCenterSvrCfg.LogLevel, "roomcentersvr"); err != nil {
		return err
	}

	err = router.InitAndRun(gconf.RoomCenterSvrCfg.SelfBusId,
		onRecvSSPacket,
		gconf.RoomCenterSvrCfg.BusMQAddr,
		misc.ServerRouteRules,
		gconf.RoomCenterSvrCfg.RegisterAddr,
	)
	if err != nil {
		return err
	}

	// IDL-driven ssrpc handlers (Phase A).
	srv := roomcenterv1.NewRoomCenterInnerServiceSServer(&service.RoomCenterInnerServiceImpl{}, ssrpc.DefaultMWOptions{})
	d := ssrpc.NewDispatcher()
	roomcenterv1.RegisterRoomCenterInnerServiceToDispatcher(d, srv)
	d.RegisterToTransactionMgr(globals.TransMgr)
	transShardCount := gconf.RoomCenterSvrCfg.TransShardCount
	if transShardCount <= 0 {
		transShardCount = transaction.DefaultShardCount()
	}
	globals.TransMgr.InitAndRunWithConfig(transaction.TransactionMgrConfig{
		MaxTrans:         misc.MaxTransNumber,
		ShardCount:       transShardCount,
		MaxPendingPerKey: 200,
	})
	logger.Infof("roomcentersvr transmgr shards=%d serial_key=routerid_or_uid", transShardCount)
	if id.IDGen, err = idgen.NewIDGen(); err != nil {
		return err
	}

	//remote loading gameconf
	if gconf.RoomCenterSvrCfg.NacosConf.IPAddr != "" {
		logger.Infof("Loading remote gameconf by Nacos group: %v ", gconf.RoomCenterSvrCfg.NacosConf.GroupName)
		err = gamedata.InitNet(net_conf.NewNacosConfigClient(gconf.RoomCenterSvrCfg.NacosConf), gconf.RoomCenterSvrCfg.NacosConf.GroupName)
		if err != nil {
			return err
		}
	}

	safego.Go(func() {
		room_ai.OnAiInitRoom()
	})

	logger.RegisterCmdBacklist(uint32(pb.CMD_ROOM_CENTER_INNER_TICK_REQ))
	logger.Infof("roomcenter svr init success")
	return globals.RoomListMgr.Init()
}

func (a *RoomMgrSvrImpl) OnReload() error {
	err := marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.RoomCenterSvrCfg)
	if err != nil {
		logger.Errorf("failed to load svr config | %s", err)
		return err
	}

	//local loading gameconf
	if gconf.RoomCenterSvrCfg.GameDataDir != "" {
		logger.Infof("Loading local file by gameconf_dir: %v ", gconf.RoomCenterSvrCfg.GameDataDir)
		err = gamedata.InitLocal(gconf.RoomCenterSvrCfg.GameDataDir)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* @Description:  proc
* @return: bool
* @Author: Iori
* @Date: 2022-04-27 21:05:01
**/
func (self *RoomMgrSvrImpl) OnProc() bool {
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
func (self *RoomMgrSvrImpl) OnTick(lastMs, nowMs int64) {
	//logger.Debugf("OnTick %v", nowMs)

	safego.Go(func() {
		globals.RoomListMgr.Tick(nowMs)
	})
}

/**
* @Description: main exit
* @Author: Iori
* @Date: 2022-04-27 21:05:07
**/
func (self *RoomMgrSvrImpl) OnExit() {
	// game exit todo something
	logger.Flush()
	logger.Infof("service exit, right now !")
	logger.Infof("================== MainSvrImpl Stop =========================")
}

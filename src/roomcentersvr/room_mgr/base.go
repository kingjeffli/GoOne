package room_mgr

import (
	roomcenterv1 "github.com/Iori372552686/GoOne/api/gen/game/roomcenter/v1"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/datetime"
	"github.com/Iori372552686/GoOne/lib/service/bus"
	"github.com/Iori372552686/GoOne/src/roomcentersvr/room_mgr/texas_room"
	pb "github.com/Iori372552686/game_protocol/protocol"
	"sync"
)

type RoomMgr struct {

	//public
	TexasMgr map[uint64]*texas_room.TexasRoomCenterMgr //德州游戏房间管理
	// more ... game room mgr

	//private
	isOpen    bool
	lastTick  int64
	eventTick int64
	sync.RWMutex
}

func NewRoomMgr() *RoomMgr {
	impl := &RoomMgr{}
	impl.TexasMgr = make(map[uint64]*texas_room.TexasRoomCenterMgr)
	return impl
}

func (impl *RoomMgr) Init() error {
	impl.isOpen = true
	return nil
}

func (impl *RoomMgr) Tick(nowMs int64) {
	if !impl.checkOpen() {
		return
	}

	if (nowMs - impl.lastTick) > 5*datetime.MS_PER_SECOND {
		impl.lastTick = nowMs

		for _, zone := range impl.TexasMgr {
			if zone.TexasMap == nil {
				continue
			}

			// 内部转发，使用 IDL 生成的一次性 one-way helper。
			roomcenterv1.NewRoomCenterInnerServiceClient().TickByRouterSimple(
				zone.Index,
				0,
				0,
				&pb.InnerTickReq{NowMs: nowMs, SrcBusId: bus.IpStringToInt(gconf.RoomCenterSvrCfg.SelfBusId)},
			)
		}
	}
}

// onExit, save data
func (impl *RoomMgr) Exit() {
	if !impl.checkOpen() {
		return
	}

	for _, zone := range impl.TexasMgr {
		zone.Exit()
	}
}

func (impl *RoomMgr) checkOpen() bool {
	return impl.isOpen
}

// ----------------------------------------------public----------------------------------------------

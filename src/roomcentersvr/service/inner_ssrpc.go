package service

import (
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/src/roomcentersvr/globals"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// RoomCenterInnerServiceImpl is the IDL-driven ssrpc implementation for roomcentersvr internal commands.
type RoomCenterInnerServiceImpl struct{}

func (s *RoomCenterInnerServiceImpl) Tick(ctx *ssrpc.Context, req *g1_protocol.InnerTickReq) (*emptypb.Empty, error) {
	// Keep the exact same routing semantics as the legacy adapter:
	// routerId (Rid) selects the zone manager.
	ins := globals.RoomListMgr.GetRoomMgrObj(ctx.Rid())
	if ins == nil {
		// legacy code returned ERR_ARGV; keep behavior.
		return nil, ssrpc.E(g1_protocol.ErrorCode_ERR_ARGV, "room mgr not found")
	}

	ins.Tick(req.GetNowMs())
	return nil, nil // one-way
}



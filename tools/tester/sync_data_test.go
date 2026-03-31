package tester

import (
	"github.com/Iori372552686/GoOne/tools/tester/tester_util"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
	"testing"
)

func TestSyncClientData(t *testing.T) {
	s := tester_util.NewSession(t)
	err := s.OpenAndLogin()
	if err != nil {
		return
	}
	defer s.LogoutAndClose()

	req := &g1_protocol.MallBuyPackageReq{
		ConfId: 8,
	}
	err = s.SendCmd(uint32(g1_protocol.CMD_MAIN_MALL_BUY_PACKAGE_REQ), req)
	if err != nil {
		return
	}

	legacyMsg := &g1_protocol.ScSyncUserData{}
	patchMsg := &g1_protocol.ScSyncUserDataV2{}
	cmd, err := s.WaitTillAnyCmd(map[uint32]proto.Message{
		uint32(g1_protocol.CMD_SC_SYNC_USER_DATA):    legacyMsg,
		uint32(g1_protocol.CMD_SC_SYNC_USER_DATA_V2): patchMsg,
	})
	if err != nil {
		return
	}
	switch cmd {
	case uint32(g1_protocol.CMD_SC_SYNC_USER_DATA):
		if legacyMsg.RoleInfo == nil {
			t.Fatalf("legacy sync payload missing role_info")
		}
	case uint32(g1_protocol.CMD_SC_SYNC_USER_DATA_V2):
		if patchMsg.GetFullSectionMask() == 0 && patchMsg.GetPatchSectionMask() == 0 {
			t.Fatalf("sync v2 payload missing full and patch masks")
		}
	default:
		t.Fatalf("unexpected sync cmd: 0x%x", cmd)
	}

	//rsp := &g1_protocol.BuyRsp{}
	//err = s.WaitTillCmd(uint32(g1_protocol.CMD_MAIN_BUY_RSP), rsp)
	//if err != nil {
	//	return
	//}
}

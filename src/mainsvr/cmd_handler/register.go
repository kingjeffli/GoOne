package cmd_handler

import (
	"github.com/Iori372552686/GoOne/lib/api/logger"
)

// 所有的命令字对应的go需要在这里先注册
func RegisterCmd() {
	logger.Infof("Register commands")
	// Phase A migration:
	// Login / Logout / HeartBeat are now registered via IDL-driven ssrpc wrappers in api/gen/... and
	// bound in src/mainsvr/app.go via RegisterMainC2SServiceToTransactionMgr.
	// ChangeName / ChangeIcon are now also migrated to ssrpc.
	// GM / Mall are now also migrated to ssrpc:
	// - CMD_MAIN_GM_GET_ROLE_REQ
	// - CMD_MAIN_GM_SET_ROLE_REQ
	// - CMD_MAIN_GM_ADD_ITEM_REQ
	// - CMD_MAIN_MALL_BUY_PACKAGE_REQ
	//globals.TransMgr.RegisterCmd(g1_protocol.CMD_MAIN_MALL_RECHARGE_REQ, NewRoleAdapter(MallRecharge))

	//------- 德州游戏房间操作  start--------
	// Phase A migration:
	// All Texas cmds are now migrated to ssrpc via MainC2SService (see api/proto/game/mainsvr/v1/c2s.proto).
	//------- 德州游戏房间操作  end--------
}

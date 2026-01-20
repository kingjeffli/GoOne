package cmd_handler

import (
	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/api/logger"
)

var CmdHandler *RegCmdHandler

/**
 * CmdHandler
 * @Description:
**/
type RegCmdHandler struct {
	ChMgr *cmd_handler.CmdHandlerMgr
}

/**
* @Description:  init regCmd
* @return: error
* @Author: Iori
* @Date: 2022-09-02 17:09:29
**/
func (self *RegCmdHandler) Init() error {
	self.RegRpcCmd()
	self.RegHttpCmd()
	self.RegWsCmd()
	return nil
}

/**
* @Description:
* @return: *RegCmdHandler
* @Author: Iori
* @Date: 2022-08-16 18:11:49
**/
func NewRegCmdHandler() *RegCmdHandler {
	impl := &RegCmdHandler{}
	impl.ChMgr = cmd_handler.NewCmdHandlerMgr()
	impl.ChMgr.Init(impl)

	return impl
}

/**
* @Description:  http cmd reg
* @Author: Iori
* @Date: 2022-05-07 15:58:04
**/
func (self *RegCmdHandler) RegHttpCmd() {
	logger.Infof("register http transaction commands")

	// Legacy HTTP cmd handler (safe_msg/:cmd). Prefer the IDL-driven route:
	// POST /v1/web/msg-sec-check (see api/proto/web/websvr/v1/web_api.proto).
	self.ChMgr.HttpRegister("msgSecCheck", MsgSecCheck) // msg security check
}

/**
* @Description:  websocket cmd reg
* @Author: Iori
* @Date: 2022-05-07 15:58:15
**/
func (self *RegCmdHandler) RegWsCmd() {
	logger.Infof("register websocket transaction commands")
}

/**
* @Description:  restapi rpc cmd reg
* @Author: Iori
* @Date: 2022-05-07 15:58:15
**/
func (self *RegCmdHandler) RegRpcCmd() {
	logger.Infof("register rpc transaction commands")
}

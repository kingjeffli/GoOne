package main

import (
	"strconv"

	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/api/datetime"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/src/connsvr/globals"
	"github.com/Iori372552686/GoOne/src/connsvr/login"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// wsLoginPreAuth handles CMD_MAIN_LOGIN_REQ from WebSocket clients.
//
// It performs the same pre-authentication flow that was previously inlined in
// onWebSocketPacket:
//  1. Unmarshal LoginReq
//  2. Validate account + channelId
//  3. Authenticate via account server
//  4. Register/update the WS connection
//  5. Forward the original packet to mainsvr via router
func wsLoginPreAuth(c cmd_handler.IContext, data []byte) g1_protocol.ErrorCode {
	wsCtx, ok := c.(*wsIContext)
	if !ok {
		logger.Errorf("wsLoginPreAuth: unexpected IContext type %T", c)
		return g1_protocol.ErrorCode_ERR_INTERNAL
	}

	req := &g1_protocol.LoginReq{}
	startTime := datetime.NowMs()

	if err := proto.Unmarshal(data, req); err != nil {
		logger.Errorf("wsLoginPreAuth: fail to unmarshal LoginReq | %v", err)
		return g1_protocol.ErrorCode_ERR_MARSHAL
	}

	if req.GetAccount() == "" || req.GetChannelId() == 0 {
		logger.Errorf("wsLoginPreAuth: account or ChannelId error")
		return g1_protocol.ErrorCode_ERR_ARGV
	}

	ret, accUid := login.OnCheckAuthByAccSvr(req.Account, req.Token, req.ChannelId, req.LoginType)
	duration := datetime.NowMs() - startTime
	logger.Infof("wsLoginPreAuth: CheckAuthByAccSvr spent ms: %s | ret=%v",
		strconv.FormatInt(duration, 10), ret)

	if !ret {
		return g1_protocol.ErrorCode_ERR_FAIL
	}

	// Update resolved uid/zone on the context.
	wsCtx.SetUid(accUid)
	zone := uint32(1) // TODO: derive from server config
	wsCtx.SetZone(zone)

	// Register/update the client connection mapping.
	client := globals.ConnWsSvr.UpdateClientByUid(wsCtx.Conn(), accUid, zone)

	// Forward the original (binary) packet body to mainsvr.
	router.SendMsgByConn(accUid, accUid, zone, wsCtx.Header().Cmd, 0, data, client.Ip, client.Port)

	return g1_protocol.ErrorCode_ERR_OK
}

// regWSHandlers registers all WS (CSPacket) handlers into the global Dispatcher.
func regWSHandlers() {
	logger.Infof("register WS dispatch handlers")
	globals.WSDispatcher.RegisterWS(uint32(g1_protocol.CMD_MAIN_LOGIN_REQ), wsLoginPreAuth)
}

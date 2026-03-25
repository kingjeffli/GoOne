package main

import (
	"net"

	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/module/misc"
	"github.com/Iori372552686/GoOne/src/connsvr/globals"
)

// proc WebSocket packet
func onWebSocketPacket(conn net.Conn, data []byte) {
	headerLen := sharedstruct.ByteLenOfCSPacketHeader()
	logger.Debugf("onWebSocketPacket: {dataLen: %v, headerLen: %v, remoteAddr: %v}\n",
		len(data), headerLen, conn.RemoteAddr().String())

	packetHeader := sharedstruct.CSPacketHeader{}
	if len(data) < packetHeader.Size() {
		logger.Errorf("Received datalen < packetHeader, packet is invalid")
		return
	}

	packetHeader.From(data)
	packetBody := data[headerLen:]
	logger.CmdDebugf(packetHeader.Cmd, "[uid: %d] Received client packet: %#v", packetHeader.Uid, packetHeader)

	if misc.IsInnerCmd(packetHeader.Cmd) {
		logger.Debugf("Received an inner command from client: %#v", packetHeader)
		return
	}

	// --- Try dispatcher: registered WS handlers (e.g. login pre-auth) ---
	ic := newWsIContext(conn, &packetHeader, globals.ConnWsSvr)
	if _, handled := globals.WSDispatcher.DispatchWS(ic, packetHeader.Cmd, packetBody); handled {
		return
	}

	// --- Default path: forward to backend server via router ---
	uid := packetHeader.Uid
	if uid == 0 {
		logger.Errorf("uid==0 and no WS handler registered for cmd %d", packetHeader.Cmd)
		return
	}

	client := globals.ConnWsSvr.GetClientByUid(uid)
	if client == nil {
		logger.Errorf("Cannot find conn by uid: %v", uid)
		return
	}

	// 前期简单测试，后期改为严谨通过rebind 与账号服验证后更新conn
	if client.Conn != conn {
		globals.ConnWsSvr.UpdateClientByUid(conn, uid, client.Zone)
	}

	router.SendMsgByConn(uid, uid, client.Zone, packetHeader.Cmd, 0, packetBody, client.Ip, client.Port)
}

// proc tcp packet
func onTcpPacket(conn net.Conn, data []byte) {
	headerLen := sharedstruct.ByteLenOfCSPacketHeader()
	logger.Debugf("OnTcpPacket: {dataLen: %v, headerLen: %v, remoteAddr: %v}\n",
		len(data), headerLen, conn.RemoteAddr())

	packetHeader := sharedstruct.CSPacketHeader{}
	if len(data) < packetHeader.Size() {
		logger.Errorf("Received datalen < packetHeader, packet is invalid")
		return
	}

	packetHeader.From(data)
	packetBody := data[headerLen:]
	logger.CmdDebugf(packetHeader.Cmd, "[uid: %d] Received client packet: %#v", packetHeader.Uid, packetHeader)

	if misc.IsInnerCmd(packetHeader.Cmd) {
		logger.Debugf("Received an inner command from client: %#v", packetHeader)
		return
	}

	// --- Try dispatcher: registered handlers (e.g. login pre-auth) ---
	ic := newWsIContext(conn, &packetHeader, globals.ConnTcpSvr)
	if _, handled := globals.WSDispatcher.DispatchWS(ic, packetHeader.Cmd, packetBody); handled {
		return
	}

	// --- Default path: forward to backend server via router ---
	uid := packetHeader.Uid
	if uid == 0 {
		logger.Errorf("uid==0 and no handler registered for cmd %d (tcp)", packetHeader.Cmd)
		return
	}

	client := globals.ConnTcpSvr.GetClientByUid(uid)
	if client == nil {
		logger.Errorf("Cannot find conn by uid: %v", uid)
		return
	}

	// 前期简单测试，后期改为严谨通过rebind 与账号服验证后更新conn
	if client.Conn != conn {
		globals.ConnTcpSvr.UpdateClientByUid(conn, uid, client.Zone)
	}

	router.SendMsgByConn(uid, uid, client.Zone, packetHeader.Cmd, 0, packetBody, client.Ip, client.Port)
}

// busMsg proc cb func
func onRecvSSPacket(packet *sharedstruct.SSPacket) {
	if misc.IsClientCmd(packet.Header.Cmd) {
		csPacketHeader := sharedstruct.CSPacketHeader{
			Uid:     packet.Header.Uid,
			Cmd:     packet.Header.Cmd,
			BodyLen: packet.Header.BodyLen,
		}
		//globals.ConnTcpSvr.SendByUid(packet.Header.Uid, csPacketHeader.ToBytes(), packet.Body)
		globals.ConnWsSvr.SendByUid(packet.Header.Uid, csPacketHeader.ToBytes(), packet.Body)
	} else {
		globals.TransMgr.ProcessSSPacket(packet)
		packet = nil // packet所有权转交给transmgr，后面不能再用packet（包括data）
	}
}

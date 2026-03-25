package main

import (
	"fmt"
	"net"

	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/net/net_mgr"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/module/misc"
	"github.com/Iori372552686/GoOne/src/connsvr/globals"
)

type clientPacketRoute struct {
	name   string
	mgr    clientConnMgr
	client *net_mgr.Client
}

func routeScore(client *net_mgr.Client, ip, port uint32) int {
	if client == nil {
		return -1
	}
	score := 0
	if ip != 0 && client.Ip == ip {
		score += 2
	}
	if port != 0 && client.Port == port {
		score++
	}
	return score
}

func pickClientPacketRoute(routes []clientPacketRoute, ip, port uint32) *clientPacketRoute {
	var (
		bestIdx   = -1
		bestScore = -1
		tied      = false
		available = -1
	)
	for i := range routes {
		if routes[i].client != nil && available == -1 {
			available = i
		}
		score := routeScore(routes[i].client, ip, port)
		if score > bestScore {
			bestIdx = i
			bestScore = score
			tied = false
			continue
		}
		if score >= 0 && score == bestScore {
			tied = true
		}
	}

	if bestScore > 0 && bestIdx >= 0 && !tied {
		return &routes[bestIdx]
	}
	if available >= 0 {
		return &routes[available]
	}
	return nil
}

func sendClientPacket(uid uint64, ip, port uint32, header []byte, body []byte) error {
	routes := []clientPacketRoute{
		{name: "ws", mgr: globals.ConnWsSvr, client: globals.ConnWsSvr.GetClientByUid(uid)},
		{name: "tcp", mgr: globals.ConnTcpSvr, client: globals.ConnTcpSvr.GetClientByUid(uid)},
	}

	route := pickClientPacketRoute(routes, ip, port)
	if route == nil {
		return fmt.Errorf("client route not found uid=%d ip=%d port=%d", uid, ip, port)
	}
	if err := route.mgr.SendByUid(uid, header, body); err != nil {
		return fmt.Errorf("send client packet via %s failed uid=%d ip=%d port=%d: %w", route.name, uid, ip, port, err)
	}
	return nil
}

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

	// --- Try dispatcher: registered client-packet handlers (e.g. login pre-auth) ---
	ic := newClientPacketIContext(conn, &packetHeader, globals.ConnWsSvr, ssrpc.TransportWS)
	if _, handled := globals.ClientPacketDispatcher.DispatchWS(ic, packetHeader.Cmd, packetBody); handled {
		return
	}

	// --- Default path: forward to backend server via router ---
	uid := packetHeader.Uid
	if uid == 0 {
		logger.Errorf("uid==0 and no client packet handler registered for cmd %d", packetHeader.Cmd)
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

	// --- Try dispatcher: registered client-packet handlers (e.g. login pre-auth) ---
	ic := newClientPacketIContext(conn, &packetHeader, globals.ConnTcpSvr, ssrpc.TransportTCP)
	if _, handled := globals.ClientPacketDispatcher.DispatchWS(ic, packetHeader.Cmd, packetBody); handled {
		return
	}

	// --- Default path: forward to backend server via router ---
	uid := packetHeader.Uid
	if uid == 0 {
		logger.Errorf("uid==0 and no client packet handler registered for cmd %d (tcp)", packetHeader.Cmd)
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
		if err := sendClientPacket(packet.Header.Uid, packet.Header.Ip, packet.Header.Flag, csPacketHeader.ToBytes(), packet.Body); err != nil {
			logger.Errorf("%v", err)
			return
		}
	} else {
		globals.TransMgr.ProcessSSPacket(packet)
		packet = nil // packet所有权转交给transmgr，后面不能再用packet（包括data）
	}
}

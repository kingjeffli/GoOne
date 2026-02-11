package main

import (
	"errors"
	"net"

	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/net/net_mgr"
	"github.com/Iori372552686/GoOne/lib/service/router"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// clientConnMgr is a minimal interface satisfied by both ConnWsTcpSvr and
// ConnTcpSvr, allowing the same wsIContext/handler to serve WS and TCP.
type clientConnMgr interface {
	UpdateClientByUid(conn net.Conn, uid uint64, zone uint32) *net_mgr.Client
	GetClientByUid(uid uint64) *net_mgr.Client
	SendByUid(uid uint64, data1 []byte, data2 []byte) error
}

// wsIContext implements cmd_handler.IContext for the WS/TCP (CSPacket) transport.
//
// It carries the raw connection and CSPacketHeader so that handlers (e.g. login
// pre-auth) can access connection-level state. After authentication the resolved
// uid/zone can be set via SetUid/SetZone for downstream use.
type wsIContext struct {
	conn    net.Conn
	uid     uint64
	zone    uint32
	header  *sharedstruct.CSPacketHeader
	connMgr clientConnMgr
}

var _ cmd_handler.IContext = (*wsIContext)(nil)

// newWsIContext creates a wsIContext from the raw connection and parsed header.
// connMgr can be either ConnWsTcpSvr (WS) or ConnTcpSvr (TCP).
func newWsIContext(conn net.Conn, header *sharedstruct.CSPacketHeader, connMgr clientConnMgr) *wsIContext {
	return &wsIContext{
		conn:    conn,
		uid:     header.Uid,
		zone:    0,
		header:  header,
		connMgr: connMgr,
	}
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func (w *wsIContext) Uid() uint64        { return w.uid }
func (w *wsIContext) Zone() uint32       { return w.zone }
func (w *wsIContext) Rid() uint64        { return 0 }
func (w *wsIContext) OriSrcBusId() uint32 { return 0 }
func (w *wsIContext) Ip() uint32         { return 0 }
func (w *wsIContext) Flag() uint32       { return 0 }

// SetUid allows a handler (e.g. login) to update the uid after authentication.
func (w *wsIContext) SetUid(uid uint64) { w.uid = uid }

// SetZone allows a handler to set the zone.
func (w *wsIContext) SetZone(zone uint32) { w.zone = zone }

// Conn returns the underlying net.Conn for connection-level operations.
func (w *wsIContext) Conn() net.Conn { return w.conn }

// Header returns the parsed CSPacketHeader.
func (w *wsIContext) Header() *sharedstruct.CSPacketHeader { return w.header }

// ConnMgr returns the connection manager for client registration.
func (w *wsIContext) ConnMgr() clientConnMgr { return w.connMgr }

// ---------------------------------------------------------------------------
// IContext implementation
// ---------------------------------------------------------------------------

// ParseMsg performs binary proto unmarshal (not JSON like the HTTP transport).
func (w *wsIContext) ParseMsg(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

// SendMsgBack serialises the response as a CSPacket and writes it to the WS
// connection. The response cmd follows the GoOne convention: request cmd + 1.
func (w *wsIContext) SendMsgBack(pbMsg proto.Message) {
	if pbMsg == nil {
		return
	}
	data, err := proto.Marshal(pbMsg)
	if err != nil {
		logger.Errorf("wsIContext.SendMsgBack marshal err=%v", err)
		return
	}
	respCmd := w.header.Cmd + 1
	csHeader := sharedstruct.CSPacketHeader{
		Uid:     w.uid,
		Cmd:     respCmd,
		BodyLen: uint32(len(data)),
	}
	if w.connMgr != nil {
		w.connMgr.SendByUid(w.uid, csHeader.ToBytes(), data)
	}
}

// ---------------------------------------------------------------------------
// RPC forwarding — delegate to router where possible, otherwise unsupported.
// ---------------------------------------------------------------------------

func (w *wsIContext) CallMsgBySvrType(svrType uint32, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	return errors.New("CallMsgBySvrType not supported in ws context")
}
func (w *wsIContext) CallMsgByRouter(svrType uint32, routerId uint64, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	return errors.New("CallMsgByRouter not supported in ws context")
}
func (w *wsIContext) CallOtherMsgBySvrType(svrType uint32, routerId, uid uint64, zone uint32, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	return errors.New("CallOtherMsgBySvrType not supported in ws context")
}

func (w *wsIContext) SendMsgByServerType(svrType uint32, cmd g1_protocol.CMD, req proto.Message) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return router.SendMsgByConn(w.uid, w.uid, w.zone, uint32(cmd), 0, data, 0, 0)
}

func (w *wsIContext) SendMsgByRouter(svrType uint32, routerId uint64, cmd g1_protocol.CMD, req proto.Message) error {
	return errors.New("SendMsgByRouter not supported in ws context")
}

// ---------------------------------------------------------------------------
// Logging — delegate to global logger.
// ---------------------------------------------------------------------------

func (w *wsIContext) Errorf(format string, args ...interface{})   { logger.Errorf(format, args...) }
func (w *wsIContext) Warningf(format string, args ...interface{}) { logger.Warningf(format, args...) }
func (w *wsIContext) Infof(format string, args ...interface{})    { logger.Infof(format, args...) }
func (w *wsIContext) Debugf(format string, args ...interface{})   { logger.Debugf(format, args...) }

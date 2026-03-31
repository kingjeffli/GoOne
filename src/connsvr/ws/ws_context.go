package ws

import (
	"errors"
	"net"

	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/net/net_mgr"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// ClientConnMgr is a minimal interface satisfied by both ConnWsTcpSvr and
// ConnTcpSvr, allowing the same ClientPacketIContext/handler to serve WS and TCP.
type ClientConnMgr interface {
	UpdateClientByUid(conn net.Conn, uid uint64, zone uint32) *net_mgr.Client
	GetClientByUid(uid uint64) *net_mgr.Client
	SendByUid(uid uint64, data1 []byte, data2 []byte) error
}

// ClientPacketIContext implements cmd_handler.IContext for the client packet
// (WebSocket / raw TCP CSPacket) transport.
//
// It carries the raw connection and CSPacketHeader so that handlers (e.g. login
// pre-auth) can access connection-level state. After authentication the resolved
// uid/zone can be set via SetUid/SetZone for downstream use.
type ClientPacketIContext struct {
	conn      net.Conn
	uid       uint64
	zone      uint32
	header    *sharedstruct.CSPacketHeader
	connMgr   ClientConnMgr
	transport ssrpc.Transport
}

var _ cmd_handler.IContext = (*ClientPacketIContext)(nil)

// newClientPacketIContext creates a ClientPacketIContext from the raw
// connection and parsed header. connMgr can be either ConnWsTcpSvr (WS) or
// ConnTcpSvr (TCP).
func NewClientPacketIContext(conn net.Conn, header *sharedstruct.CSPacketHeader, connMgr ClientConnMgr, transport ssrpc.Transport) *ClientPacketIContext {
	return &ClientPacketIContext{
		conn:      conn,
		uid:       header.Uid,
		zone:      0,
		header:    header,
		connMgr:   connMgr,
		transport: transport,
	}
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func (c *ClientPacketIContext) Uid() uint64         { return c.uid }
func (c *ClientPacketIContext) Zone() uint32        { return c.zone }
func (c *ClientPacketIContext) Rid() uint64         { return 0 }
func (c *ClientPacketIContext) OriSrcBusId() uint32 { return 0 }
func (c *ClientPacketIContext) Ip() uint32          { return 0 }
func (c *ClientPacketIContext) Flag() uint32        { return 0 }

// SetUid allows a handler (e.g. login) to update the uid after authentication.
func (c *ClientPacketIContext) SetUid(uid uint64) { c.uid = uid }

// SetZone allows a handler to set the zone.
func (c *ClientPacketIContext) SetZone(zone uint32) { c.zone = zone }

// Conn returns the underlying net.Conn for connection-level operations.
func (c *ClientPacketIContext) Conn() net.Conn { return c.conn }

// Header returns the parsed CSPacketHeader.
func (c *ClientPacketIContext) Header() *sharedstruct.CSPacketHeader { return c.header }

// ConnMgr returns the connection manager for client registration.
func (c *ClientPacketIContext) ConnMgr() ClientConnMgr { return c.connMgr }

// SSRPCTransport tells WrapWS whether this client packet arrived via WS or TCP.
func (c *ClientPacketIContext) SSRPCTransport() ssrpc.Transport { return c.transport }

// ---------------------------------------------------------------------------
// IContext implementation
// ---------------------------------------------------------------------------

// ParseMsg performs binary proto unmarshal (not JSON like the HTTP transport).
func (c *ClientPacketIContext) ParseMsg(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

// SendMsgBack serialises the response as a CSPacket and writes it to the
// current client connection. The response cmd follows the GoOne convention:
// request cmd + 1.
func (c *ClientPacketIContext) SendMsgBack(pbMsg proto.Message) {
	if pbMsg == nil {
		return
	}
	data, err := proto.Marshal(pbMsg)
	if err != nil {
		logger.Errorf("ClientPacketIContext.SendMsgBack marshal err=%v", err)
		return
	}
	respCmd := c.header.Cmd + 1
	csHeader := sharedstruct.CSPacketHeader{
		Uid:     c.uid,
		Cmd:     respCmd,
		BodyLen: uint32(len(data)),
	}
	if c.connMgr != nil {
		c.connMgr.SendByUid(c.uid, csHeader.ToBytes(), data)
	}
}

// ---------------------------------------------------------------------------
// RPC forwarding — delegate to router where possible, otherwise unsupported.
// ---------------------------------------------------------------------------

func (c *ClientPacketIContext) CallMsgBySvrType(svrType uint32, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	return errors.New("CallMsgBySvrType not supported in client packet context")
}
func (c *ClientPacketIContext) CallMsgByRouter(svrType uint32, routerId uint64, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	return errors.New("CallMsgByRouter not supported in client packet context")
}
func (c *ClientPacketIContext) CallOtherMsgBySvrType(svrType uint32, routerId, uid uint64, zone uint32, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	return errors.New("CallOtherMsgBySvrType not supported in client packet context")
}

func (c *ClientPacketIContext) SendMsgByServerType(svrType uint32, cmd g1_protocol.CMD, req proto.Message) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return router.SendMsgByConn(c.uid, c.uid, c.zone, uint32(cmd), 0, data, 0, 0)
}

func (c *ClientPacketIContext) SendMsgByRouter(svrType uint32, routerId uint64, cmd g1_protocol.CMD, req proto.Message) error {
	return errors.New("SendMsgByRouter not supported in client packet context")
}

// ---------------------------------------------------------------------------
// Logging — delegate to global logger.
// ---------------------------------------------------------------------------

func (c *ClientPacketIContext) Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}
func (c *ClientPacketIContext) Warningf(format string, args ...interface{}) {
	logger.Warningf(format, args...)
}
func (c *ClientPacketIContext) Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}
func (c *ClientPacketIContext) Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

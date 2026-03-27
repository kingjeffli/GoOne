package transaction

import (
	"sync/atomic"
	"time"

	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/Iori372552686/GoOne/lib/service/bus"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

type TransactionMgr struct {
	started     bool
	cmdHandlers map[g1_protocol.CMD]cmd_handler.CmdHandlerFunc

	config       TransactionMgrConfig
	useSerialKey bool
	maxTransNum  int32
	shards       []*transactionShard
	roundRobin   atomic.Uint32

	activeTransactions atomic.Int64
	pendingPackets     atomic.Int64
	droppedPackets     atomic.Int64
}

type transactionShard struct {
	mgr   *TransactionMgr
	index int

	chanInPacket chan *sharedstruct.SSPacket
	chanTransRet chan uint32

	nextTransID    uint32
	transIDStep    uint32
	transMap       map[uint32]*Transaction
	keyInProcess   map[uint64]bool
	pendingPackets map[uint64][]*sharedstruct.SSPacket
}

func (m *TransactionMgr) InitAndRun(maxTrans int32, useUidLock bool, maxUidPendingPacket int) {
	m.InitAndRunWithConfig(TransactionMgrConfig{
		MaxTrans:         maxTrans,
		ShardCount:       1,
		MaxPendingPerKey: maxUidPendingPacket,
	})
	m.useSerialKey = useUidLock
}

func (m *TransactionMgr) InitAndRunWithConfig(cfg TransactionMgrConfig) {
	if m.started {
		logger.Errorf("transmgr can only be InitAndRun once")
		return
	}

	cfg = normalizeConfig(cfg)
	m.started = true
	m.config = cfg
	m.useSerialKey = true
	m.maxTransNum = cfg.MaxTrans

	shardBufSize := perShardBufferSize(cfg.MaxTrans, cfg.ShardCount)
	m.shards = make([]*transactionShard, 0, cfg.ShardCount)
	for i := 0; i < cfg.ShardCount; i++ {
		shard := &transactionShard{
			mgr:            m,
			index:          i,
			chanInPacket:   make(chan *sharedstruct.SSPacket, shardBufSize),
			chanTransRet:   make(chan uint32, shardBufSize),
			nextTransID:    uint32(i + 1),
			transIDStep:    uint32(cfg.ShardCount),
			transMap:       make(map[uint32]*Transaction, shardBufSize),
			keyInProcess:   make(map[uint64]bool),
			pendingPackets: make(map[uint64][]*sharedstruct.SSPacket),
		}
		m.shards = append(m.shards, shard)
		go shard.run()
	}
}

func (m *TransactionMgr) RegisterCmd(cmd g1_protocol.CMD, cmdHandler cmd_handler.CmdHandlerFunc) {
	if m.started {
		logger.Fatalf("RegisterCmd must be invoked before InitAndRun")
	}

	if m.cmdHandlers == nil {
		m.cmdHandlers = make(map[g1_protocol.CMD]cmd_handler.CmdHandlerFunc)
	}
	m.cmdHandlers[cmd] = cmdHandler
}

func (m *TransactionMgr) ProcessSSPacket(packet *sharedstruct.SSPacket) {
	if packet == nil {
		return
	}
	shard := m.selectShard(packet)
	if shard == nil {
		logger.Errorf("transmgr is not initialized, drop packet {header:%#v}", packet.Header)
		return
	}
	shard.chanInPacket <- packet
}

func (m *TransactionMgr) SendPbMsgToMyself(selfBusId uint32, rid uint64, uid uint64, zone uint32, cmd g1_protocol.CMD, pbMsg proto.Message) {
	data, err := proto.Marshal(pbMsg)
	if err != nil {
		logger.Errorf("Failed to SendMsgToMyself {uid:%v, cmd,%v, msg:%v}", uid, cmd, pbMsg)
		return
	}

	packet := &sharedstruct.SSPacket{
		Header: sharedstruct.SSPacketHeader{
			SrcBusID:   selfBusId,
			DstBusID:   selfBusId,
			SrcTransID: 0,
			DstTransID: 0,
			Uid:        uid,
			RouterID:   rid,
			Cmd:        uint32(cmd),
			Zone:       zone,
			Ip:         0,
			Flag:       0,
			BodyLen:    uint32(len(data)),
			CmdSeq:     0,
		},
		Body: data,
	}

	m.ProcessSSPacket(packet)
}

func (m *TransactionMgr) StatsSnapshot() TransactionMgrStats {
	return TransactionMgrStats{
		ShardCount:         len(m.shards),
		ActiveTransactions: m.activeTransactions.Load(),
		PendingPackets:     m.pendingPackets.Load(),
		DroppedPackets:     m.droppedPackets.Load(),
	}
}

func (m *TransactionMgr) selectShard(packet *sharedstruct.SSPacket) *transactionShard {
	if len(m.shards) == 0 {
		return nil
	}
	if len(m.shards) == 1 {
		return m.shards[0]
	}

	if packet.Header.DstTransID != 0 {
		idx := int((packet.Header.DstTransID - 1) % uint32(len(m.shards)))
		return m.shards[idx]
	}

	if key, ok := m.serialKeyFromHeader(packet.Header); ok {
		idx := int(key % uint64(len(m.shards)))
		return m.shards[idx]
	}

	next := m.roundRobin.Add(1)
	idx := int((next - 1) % uint32(len(m.shards)))
	return m.shards[idx]
}

func (m *TransactionMgr) serialKeyFromHeader(header sharedstruct.SSPacketHeader) (uint64, bool) {
	if !m.useSerialKey {
		return 0, false
	}
	if header.RouterID != 0 {
		return header.RouterID, true
	}
	if header.Uid != 0 {
		return header.Uid, true
	}
	return 0, false
}

func (m *TransactionMgr) tryAcquireTransSlot() bool {
	for {
		cur := m.activeTransactions.Load()
		if cur >= int64(m.maxTransNum) {
			return false
		}
		if m.activeTransactions.CompareAndSwap(cur, cur+1) {
			return true
		}
	}
}

func (m *TransactionMgr) releaseTransSlot() {
	m.activeTransactions.Add(-1)
}

func (m *TransactionMgr) onPendingPacketAdded() {
	m.pendingPackets.Add(1)
}

func (m *TransactionMgr) onPendingPacketRemoved() {
	m.pendingPackets.Add(-1)
}

func (m *TransactionMgr) onPacketDropped() {
	m.droppedPackets.Add(1)
}

func (s *transactionShard) run() {
	for {
		select {
		case packet, ok := <-s.chanInPacket:
			if !ok {
				logger.Error("transaction shard chanInPacket is closed")
				return
			}
			s.processSSPacket(packet)
		case transID, ok := <-s.chanTransRet:
			if !ok {
				logger.Error("transaction shard chanTransRet is closed")
				return
			}
			s.processTransactionRet(transID)
		}
	}
}

func (s *transactionShard) processSSPacket(packet *sharedstruct.SSPacket) int32 {
	uid := packet.Header.Uid
	rid := packet.Header.RouterID
	dstTransID := packet.Header.DstTransID
	cmd := packet.Header.Cmd
	logger.CmdDebugf(cmd, "Recv uid: %v | SrcBusID: %v | cmd [%v]", uid, bus.IpIntToString(packet.Header.SrcBusID), g1_protocol.CMD(packet.Header.Cmd))

	if dstTransID != 0 {
		trans, in := s.transMap[dstTransID]
		if !in {
			logger.Errorf("received a response can't be handled by any transaction{header:%#v}", packet.Header)
			return -3
		}
		if !packet.SendToChan(trans.chanIn, 3*time.Second) {
			logger.Errorf("timeout to send message to transaction {header: %#v}", packet.Header)
			return -4
		}
		return 0
	}

	serialKey, hasSerialKey := s.mgr.serialKeyFromHeader(packet.Header)
	if hasSerialKey && s.keyInProcess[serialKey] {
		packets := s.pendingPackets[serialKey]
		if len(packets) >= s.mgr.config.MaxPendingPerKey {
			s.mgr.onPacketDropped()
			logger.Errorf("Drop a packet for serial key {key:%d, uid:%d, rid:%d, cmd:%d}", serialKey, uid, rid, cmd)
			return -1
		}

		s.pendingPackets[serialKey] = append(packets, packet)
		s.mgr.onPendingPacketAdded()
		return 0
	}

	cmdHandler, in := s.mgr.cmdHandlers[g1_protocol.CMD(cmd)]
	if !in {
		logger.Errorf("no reg cmd {cmd:0x%x}", cmd)
		return -2
	}

	if !s.mgr.tryAcquireTransSlot() {
		logger.Errorf("reach transaction count limit {max:%v, packetHeader:%v}", s.mgr.maxTransNum, packet.Header)
		return -5
	}

	transID := s.nextTransID
	s.nextTransID += s.transIDStep

	transaction := newTransaction(transID, packet.Header, make(chan *sharedstruct.SSPacket, 1))
	s.transMap[transID] = transaction
	if hasSerialKey {
		s.keyInProcess[serialKey] = true
	}

	go transaction.run(cmdHandler, packet, s.chanTransRet)
	return 0
}

func (s *transactionShard) processTransactionRet(transID uint32) {
	trans, in := s.transMap[transID]
	if !in || trans == nil {
		logger.Errorf("no trans in map {transId:%d}", transID)
		return
	}

	close(trans.chanIn)
	delete(s.transMap, transID)
	s.mgr.releaseTransSlot()

	serialKey, hasSerialKey := s.mgr.serialKeyFromHeader(trans.OriPacketHeader)
	if !hasSerialKey {
		return
	}

	delete(s.keyInProcess, serialKey)

	packets, in := s.pendingPackets[serialKey]
	if !in {
		return
	}
	if len(packets) == 0 {
		delete(s.pendingPackets, serialKey)
		return
	}

	nextPacket := packets[0]
	if len(packets) == 1 {
		delete(s.pendingPackets, serialKey)
	} else {
		s.pendingPackets[serialKey] = packets[1:]
	}
	s.mgr.onPendingPacketRemoved()
	s.processSSPacket(nextPacket)
}

func perShardBufferSize(maxTrans int32, shardCount int) int {
	if shardCount <= 0 {
		return 1
	}

	size := int(maxTrans) / shardCount
	if int(maxTrans)%shardCount != 0 {
		size++
	}
	if size <= 0 {
		return 1
	}
	return size
}

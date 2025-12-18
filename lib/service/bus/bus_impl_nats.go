package bus

import (
	"fmt"
	"strings"
	"time"

	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/nats-io/nats.go"
)

type BusImplNatsMQ struct {
	selfBusId uint32
	timeout   time.Duration
	chanOut   chan outMsg
	chanIn    chan []byte
	onRecv    MsgHandler

	url           string
	subjectPrefix string
	queueGroup    string
}

func NewBusImplNatsMQ(selfBusId uint32, onRecvMsg MsgHandler, conf NatsConfig) IBus {
	prefix := strings.TrimSpace(conf.SubjectPrefix)
	if prefix == "" {
		prefix = "bus"
	}
	impl := &BusImplNatsMQ{
		selfBusId:      selfBusId,
		timeout:        3 * time.Second,
		chanOut:        make(chan outMsg, 10000),
		chanIn:         make(chan []byte, 10000),
		onRecv:         onRecvMsg,
		url:            strings.TrimSpace(conf.URL),
		subjectPrefix:  prefix,
		queueGroup:     strings.TrimSpace(conf.QueueGroup),
	}
	go impl.run()
	return impl
}

func (b *BusImplNatsMQ) SelfBusId() uint32 { return b.selfBusId }
func (b *BusImplNatsMQ) SetReceiver(onRecvMsg MsgHandler) { b.onRecv = onRecvMsg }

func (b *BusImplNatsMQ) subjectFor(busId uint32) string {
	return b.subjectPrefix + "." + calcQueueName(busId)
}

func (b *BusImplNatsMQ) Send(dstBusId uint32, data1 []byte, data2 []byte) error {
	header := busPacketHeader{}
	header.version = 0
	header.passCode = passCode
	header.srcBusId = b.SelfBusId()
	header.dstBusId = dstBusId

	msg := outMsg{}
	msg.busId = dstBusId
	msg.topics = b.subjectFor(dstBusId)
	msg.data = make([]byte, byteLenOfBusPacketHeader()+len(data1)+len(data2))
	pos := 0
	header.To(msg.data[pos:])
	pos += byteLenOfBusPacketHeader()
	copy(msg.data[pos:], data1)
	pos += len(data1)
	if data2 != nil && len(data2) > 0 {
		copy(msg.data[pos:], data2)
		pos += len(data2)
	}

	if !sendToMsgChan(b.chanOut, msg, b.timeout) {
		return fmt.Errorf("nats bus.chanOut<-msg time out")
	}
	return nil
}

func (b *BusImplNatsMQ) process() error {
	if b.url == "" {
		return fmt.Errorf("nats url is empty")
	}
	nc, err := nats.Connect(
		b.url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return err
	}
	defer nc.Close()

	mySubject := b.subjectFor(b.selfBusId)
	logger.Infof("NATS bus connected, subscribe: %s", mySubject)

	var sub *nats.Subscription
	if b.queueGroup != "" {
		sub, err = nc.QueueSubscribe(mySubject, b.queueGroup, func(m *nats.Msg) {
			buf := make([]byte, len(m.Data))
			copy(buf, m.Data)
			select {
			case b.chanIn <- buf:
			default:
			}
		})
	} else {
		sub, err = nc.Subscribe(mySubject, func(m *nats.Msg) {
			buf := make([]byte, len(m.Data))
			copy(buf, m.Data)
			select {
			case b.chanIn <- buf:
			default:
			}
		})
	}
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	_ = nc.Flush()

	for {
		select {
		case msgOut, ok := <-b.chanOut:
			if !ok {
				return fmt.Errorf("chanOut of bus is closed")
			}
			if err := nc.Publish(msgOut.topics, msgOut.data); err != nil {
				logger.Errorf("Failed to publish nats message {subject:%v, dataLen:%v}| %v", msgOut.topics, len(msgOut.data), err)
			}
		case data, ok := <-b.chanIn:
			if !ok {
				return fmt.Errorf("chanIn of bus is closed")
			}
			if len(data) < byteLenOfBusPacketHeader() {
				continue
			}
			header := busPacketHeader{}
			header.From(data)
			if header.passCode != passCode {
				logger.Warningf("Received a bus message with wrong pass code: %#v", header)
				continue
			}
			if b.onRecv != nil {
				recvData := make([]byte, len(data)-byteLenOfBusPacketHeader())
				copy(recvData, data[byteLenOfBusPacketHeader():])
				b.onRecv(header.srcBusId, recvData)
			}
		}
	}
}

func (b *BusImplNatsMQ) run() {
	retryCount := 0
	for {
		processStartTime := time.Now()
		err := b.process()
		if time.Since(processStartTime) > time.Minute {
			retryCount = 0
		}
		retryCount++
		retryAfterSeconds := (retryCount - 1) * 2
		if retryAfterSeconds > 30 {
			retryAfterSeconds = 30
		}
		logger.Errorf("Error occur in processing bus(nats). Retry later {retryTimes: %v, afterSeconds:%v} | %v",
			retryCount, retryAfterSeconds, err)
		time.Sleep(time.Duration(retryAfterSeconds) * time.Second)
	}
}



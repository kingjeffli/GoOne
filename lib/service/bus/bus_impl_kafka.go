package bus

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/segmentio/kafka-go"
)

type BusImplKafkaMQ struct {
	selfBusId uint32
	timeout   time.Duration
	chanOut   chan outMsg
	chanIn    chan []byte
	onRecv    MsgHandler

	brokers       []string
	topicPrefix   string
	groupIDPrefix string
	stopCh        chan struct{}
	closed        atomic.Bool
	closeOnce     sync.Once
}

func NewBusImplKafkaMQ(selfBusId uint32, onRecvMsg MsgHandler, conf KafkaConfig) IBus {
	topicPrefix := strings.TrimSpace(conf.TopicPrefix)
	if topicPrefix == "" {
		topicPrefix = "bus"
	}
	groupPrefix := strings.TrimSpace(conf.GroupIDPrefix)
	if groupPrefix == "" {
		groupPrefix = "bus"
	}
	impl := &BusImplKafkaMQ{
		selfBusId:     selfBusId,
		timeout:       3 * time.Second,
		chanOut:       make(chan outMsg, 10000),
		chanIn:        make(chan []byte, 10000),
		onRecv:        onRecvMsg,
		brokers:       conf.Brokers,
		topicPrefix:   topicPrefix,
		groupIDPrefix: groupPrefix,
		stopCh:        make(chan struct{}),
	}
	go impl.run()
	return impl
}

func (b *BusImplKafkaMQ) SelfBusId() uint32                { return b.selfBusId }
func (b *BusImplKafkaMQ) SetReceiver(onRecvMsg MsgHandler) { b.onRecv = onRecvMsg }

func (b *BusImplKafkaMQ) topicFor(busId uint32) string {
	return b.topicPrefix + "." + calcQueueName(busId)
}

func (b *BusImplKafkaMQ) groupFor(busId uint32) string {
	return b.groupIDPrefix + "." + calcQueueName(busId)
}

func (b *BusImplKafkaMQ) Send(dstBusId uint32, data1 []byte, data2 []byte) error {
	if b.closed.Load() {
		return ErrBusClosed
	}
	header := busPacketHeader{}
	header.version = 0
	header.passCode = passCode
	header.srcBusId = b.SelfBusId()
	header.dstBusId = dstBusId

	msg := outMsg{}
	msg.busId = dstBusId
	msg.topics = b.topicFor(dstBusId)
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
		return fmt.Errorf("kafka bus.chanOut<-msg time out")
	}
	return nil
}

func (b *BusImplKafkaMQ) Close() error {
	b.closeOnce.Do(func() {
		b.closed.Store(true)
		close(b.stopCh)
	})
	return nil
}

func (b *BusImplKafkaMQ) process() error {
	if len(b.brokers) == 0 {
		return fmt.Errorf("kafka brokers is empty")
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(b.brokers...),
		RequiredAcks: kafka.RequireOne,
		Balancer:     &kafka.LeastBytes{},
		Async:        false,
	}
	defer w.Close()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  b.brokers,
		Topic:    b.topicFor(b.selfBusId),
		GroupID:  b.groupFor(b.selfBusId),
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	ctx := context.Background()

	readCtx, cancelRead := context.WithCancel(ctx)
	defer cancelRead()
	go func() {
		for {
			m, err := r.ReadMessage(readCtx)
			if err != nil {
				return
			}
			buf := make([]byte, len(m.Value))
			copy(buf, m.Value)
			select {
			case b.chanIn <- buf:
			case <-b.stopCh:
				return
			}
		}
	}()

	for {
		select {
		case <-b.stopCh:
			return nil
		case msgOut, ok := <-b.chanOut:
			if !ok {
				return fmt.Errorf("chanOut of bus is closed")
			}
			if err := w.WriteMessages(ctx, kafka.Message{
				Topic: msgOut.topics,
				Value: msgOut.data,
			}); err != nil {
				logger.Errorf("Failed to publish kafka message {topic:%v, dataLen:%v}| %v", msgOut.topics, len(msgOut.data), err)
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

func (b *BusImplKafkaMQ) run() {
	retryCount := 0
	for {
		if b.closed.Load() {
			return
		}
		processStartTime := time.Now()
		err := b.process()
		if b.closed.Load() {
			return
		}
		if time.Since(processStartTime) > time.Minute {
			retryCount = 0
		}
		retryCount++
		retryAfterSeconds := (retryCount - 1) * 2
		if retryAfterSeconds > 30 {
			retryAfterSeconds = 30
		}
		logger.Errorf("Error occur in processing bus(kafka). Retry later {retryTimes: %v, afterSeconds:%v} | %v",
			retryCount, retryAfterSeconds, err)
		select {
		case <-b.stopCh:
			return
		case <-time.After(time.Duration(retryAfterSeconds) * time.Second):
		}
	}
}

func init() {
	RegisterBus("kafka", func(selfBusId uint32, onRecvMsg MsgHandler, conf any) (IBus, error) {
		cfg, ok := conf.(KafkaConfig)
		if !ok {
			return nil, fmt.Errorf("kafka arg must be KafkaConfig")
		}
		return NewBusImplKafkaMQ(selfBusId, onRecvMsg, cfg), nil
	})
}

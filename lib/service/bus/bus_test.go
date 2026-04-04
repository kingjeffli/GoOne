package bus

import (
	"log"
	"os"
	"testing"
	"time"
)

func onRecvMsg(srcBusID uint32, data []byte) error {
	log.Printf("srcBusID:%v, data:%v", srcBusID, data)

	return nil
}

func TestRabbitMQBus(t *testing.T) {
	if os.Getenv("BUS_ITEST") != "1" {
		t.Skip("set BUS_ITEST=1 to run bus integration tests")
	}
	impl, err := CreateBus(IpStringToInt("1.1.2.2"), onRecvMsg, "amqp://guest:guest@127.0.0.1:5672/")
	if err != nil || impl == nil {
		t.Skip("rabbitmq bus not available or CreateBus returned nil")
	}

	if err := impl.Send(impl.SelfBusId(), []byte("abc"), nil); err != nil {
		t.Logf("rabbitmq Send error: %v", err)
	}

	time.Sleep(2 * time.Second)
}

func TestNSQBus(t *testing.T) {
	if os.Getenv("BUS_ITEST") != "1" {
		t.Skip("set BUS_ITEST=1 to run bus integration tests")
	}
	conf := NSQConfig{
		LookupAddrs: []string{"127.0.0.1:4161"},
		NsqdAddr:    "127.0.0.1:4150",
		TopicPrefix: "test",
		Channel:     "ch",
		Concurrency: 1,
	}

	impl := NewBusImplNsqMQ(1, onRecvMsg, conf)
	if impl == nil {
		t.Skip("nsq bus not available or NewBusImplNsqMQ returned nil")
	}

	if err := impl.SendTo("test", []byte("abc"), []byte("123")); err != nil {
		t.Logf("nsq SendTo error: %v", err)
	}

	time.Sleep(2 * time.Second)
}

func TestNatsBus(t *testing.T) {
	if os.Getenv("BUS_ITEST") != "1" {
		t.Skip("set BUS_ITEST=1 to run bus integration tests")
	}
	conf := NatsConfig{
		URL:           "nats://127.0.0.1:4222",
		SubjectPrefix: "testbus",
		QueueGroup:    "test-group",
	}
	implAny, err := createBusByTypeE("nats", 1, onRecvMsg, conf)
	if err != nil || implAny == nil {
		t.Skipf("nats bus not available: %v", err)
	}

	impl := implAny
	if err := impl.Send(impl.SelfBusId(), []byte("abc"), nil); err != nil {
		t.Logf("nats Send error: %v", err)
	}

	time.Sleep(2 * time.Second)
}

func TestKafkaBus(t *testing.T) {
	if os.Getenv("BUS_ITEST") != "1" {
		t.Skip("set BUS_ITEST=1 to run bus integration tests")
	}
	conf := KafkaConfig{
		Brokers:       []string{"127.0.0.1:9092"},
		TopicPrefix:   "testbus",
		GroupIDPrefix: "testgroup",
	}
	implAny, err := createBusByTypeE("kafka", 1, onRecvMsg, conf)
	if err != nil || implAny == nil {
		t.Skipf("kafka bus not available: %v", err)
	}

	impl := implAny
	if err := impl.Send(impl.SelfBusId(), []byte("abc"), nil); err != nil {
		t.Logf("kafka Send error: %v", err)
	}

	time.Sleep(2 * time.Second)
}

func TestRocketMQBus(t *testing.T) {
	if os.Getenv("BUS_ITEST") != "1" {
		t.Skip("set BUS_ITEST=1 to run bus integration tests")
	}
	conf := RocketMQConfig{
		NameServers:   []string{"127.0.0.1:9876"},
		Topic:         "testbus",
		ConsumerGroup: "testbus_group",
	}
	implAny, err := createBusByTypeE("rocketmq", 1, onRecvMsg, conf)
	if err != nil || implAny == nil {
		t.Skipf("rocketmq bus not available: %v", err)
	}

	impl := implAny
	if err := impl.Send(impl.SelfBusId(), []byte("abc"), nil); err != nil {
		t.Logf("rocketmq Send error: %v", err)
	}

	time.Sleep(2 * time.Second)
}

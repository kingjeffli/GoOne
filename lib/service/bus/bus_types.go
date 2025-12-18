package bus

// RabbitMQConfig for RabbitMQ bus backend.
type RabbitMQConfig struct {
	Addr string `json:"addr" yaml:"addr"`
}

// NSQConfig for NSQ bus backend.
type NSQConfig struct {
	LookupAddrs []string `json:"lookup_addrs" yaml:"lookup_addrs"`
	NsqdAddr    string   `json:"nsqd_addr" yaml:"nsqd_addr"` // host:port
	TopicPrefix string   `json:"topic_prefix" yaml:"topic_prefix"`
	Channel     string   `json:"channel" yaml:"channel"`
	Concurrency int      `json:"concurrency" yaml:"concurrency"`
}

// NatsConfig for NATS bus backend.
type NatsConfig struct {
	URL           string `json:"url" yaml:"url"` // nats://host:port
	SubjectPrefix string `json:"subject_prefix" yaml:"subject_prefix"`
	QueueGroup    string `json:"queue_group" yaml:"queue_group"`
}

// KafkaConfig for Kafka bus backend.
type KafkaConfig struct {
	Brokers       []string `json:"brokers" yaml:"brokers"`
	TopicPrefix   string   `json:"topic_prefix" yaml:"topic_prefix"`
	GroupIDPrefix string   `json:"group_id_prefix" yaml:"group_id_prefix"`
}

// RocketMQConfig for RocketMQ bus backend.
type RocketMQConfig struct {
	NameServers   []string `json:"name_servers" yaml:"name_servers"`
	Topic         string   `json:"topic" yaml:"topic"`
	ConsumerGroup string   `json:"consumer_group" yaml:"consumer_group"`
}



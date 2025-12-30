package gconf

import (
	"flag"
	"github.com/Iori372552686/GoOne/lib/web/web_gin"

	"github.com/Iori372552686/GoOne/lib/api/http_sign"
	"github.com/Iori372552686/GoOne/lib/api/net_conf"
	"github.com/Iori372552686/GoOne/lib/api/rest_api"
	"github.com/Iori372552686/GoOne/lib/db/redis"
	orm "github.com/Iori372552686/GoOne/lib/db/xorm"
)

var SvrConfFile = flag.String("svr_conf", "../commconf/server_conf.yaml", "app config yaml file")

type BaseCfg struct {
	// ParseConfig parses registry address strings like:
	//   - "127.0.0.1:2181"                       (defaults to zk)
	//   - "zk://127.0.0.1:2181?root=/&service=online&timeout=30s"
	//   - "etcd://127.0.0.1:2379,127.0.0.2:2379?namespace=/microservices&service=online&timeout=5s"
	//   - "consul://127.0.0.1:8500?service=online&healthcheck=true&heartbeat=true&health_interval=10"
	//   - "nacos://127.0.0.1:8848?service=online&group=DEFAULT_GROUP&cluster=DEFAULT&kind=grpc&weight=100"
	//   - "k8s://?service=online&incluster=true"
	RegisterAddr string `yaml:"register_addr"` // registry/register 地址

	// ParseAddr parses a single bus addr string into (implType, backendConfig).
	// Supported examples:
	// - amqp://guest:guest@127.0.0.1:5672/                         (rabbitmq)
	// - rabbitmq://?addr=amqp://guest:guest@127.0.0.1:5672/        (rabbitmq)
	// - nats://127.0.0.1:4222?subject_prefix=bus&queue_group=g1    (nats)
	// - kafka://127.0.0.1:9092,127.0.0.2:9092?topic_prefix=bus     (kafka)
	// - rocketmq://127.0.0.1:9876?topic=goone_bus&consumer_group=goone_bus  (rocketmq)
	// - nsq://127.0.0.1:4150?lookup=127.0.0.1:4161&topics=test&chan=ch&concurrency=3 (nsq)
	BusMQAddr          string             `yaml:"bus_mq_addr"`          // bus mq 地址
	GameDataDir        string             `yaml:"game_data_dir"`        // 游戏数据目录
	SensitiveWordsFile string             `yaml:"sensitive_words_file"` // 敏感词文件
	NacosConf          net_conf.NacosConf `yaml:"nacos_conf"`           // nacos配置
	OrmConf            []orm.Config       `yaml:"orm_instances"`        // mysql配置
	HTTPSigns          []http_sign.Config `yaml:"http_sign"`            // http签名配置
	RestApiConf        []rest_api.Config  `yaml:"rest_api_config"`      // restapi配置
	DbInstances        []redis.Config     `yaml:"db_instances"`         // redis配置
	Pprof              bool               `yaml:"pprof"`                // 是否开启pprof
}

type ConnSvr struct {
	SelfBusId  string `yaml:"self_bus_id"`
	ListenPort int    `yaml:"listen_port"`
	LogDir     string `yaml:"log_dir"`
	LogLevel   string `yaml:"log_level"`
}

type InfoSvr struct {
	SelfBusId string `yaml:"self_bus_id"`
	LogDir    string `yaml:"log_dir"`
	LogLevel  string `yaml:"log_level"`
}

type MainSvr struct {
	SelfBusId string `yaml:"self_bus_id"`
	LogDir    string `yaml:"log_dir"`
	LogLevel  string `yaml:"log_level"`
}

type MySqlSvr struct {
	SelfBusId string `yaml:"self_bus_id"`
	LogDir    string `yaml:"log_dir"`
	LogLevel  string `yaml:"log_level"`
}

type RoomCenterSvr struct {
	SelfBusId string `yaml:"self_bus_id"`
	LogDir    string `yaml:"log_dir"`
	LogLevel  string `yaml:"log_level"`
}

type TexasSvr struct {
	SelfBusId string `yaml:"self_bus_id"`
	LogDir    string `yaml:"log_dir"`
	LogLevel  string `yaml:"log_level"`
}

type WebSvr struct {
	SelfBusId string `yaml:"self_bus_id"`
	LogDir    string `yaml:"log_dir"`
	LogLevel  string `yaml:"log_level"`

	HttpServer web_gin.Config `json:"http_server" yaml:"http_server"`
}

// connsvr配置
type ConnConfig struct {
	BaseCfg `yaml:"base_cfg"`
	ConnSvr `yaml:"connsvr"`
}

var ConnSvrCfg ConnConfig

// infosvr配置
type InfoConfig struct {
	BaseCfg `yaml:"base_cfg"`
	InfoSvr `yaml:"infosvr"`
}

var InfoSvrCfg InfoConfig

// mainsvr配置
type MainSvrConfig struct {
	BaseCfg `yaml:"base_cfg"`
	MainSvr `yaml:"mainsvr"`
}

var MainSvrCfg MainSvrConfig

// mysqlsvr配置
type MySqlSvrConfig struct {
	BaseCfg  `yaml:"base_cfg"`
	MySqlSvr `yaml:"mysqlsvr"`
}

var MySqlSvrCfg MySqlSvrConfig

// roomcentersvr配置
type RoomCenterConfig struct {
	BaseCfg       `yaml:"base_cfg"`
	RoomCenterSvr `yaml:"roomcentersvr"`
}

var RoomCenterSvrCfg RoomCenterConfig

// websvr配置
type webSvrConfig struct {
	BaseCfg `yaml:"base_cfg"`
	WebSvr  `yaml:"websvr"`
}

var WebSvrCfg webSvrConfig

// texassvr配置
type TexasConfig struct {
	BaseCfg  `yaml:"base_cfg"`
	TexasSvr `yaml:"texassvr"`
}

var TexasSvrCfg TexasConfig

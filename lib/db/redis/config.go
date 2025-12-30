package redis

// db config struct
type Config struct {
	InstanceID  int    `json:"instance_id" yaml:"instance_id"`
	IP          string `json:"ip" yaml:"ip"`
	Port        int    `json:"port" yaml:"port"`
	Password    string `json:"password" yaml:"password"`
	IsCluster   bool   `json:"is_cluster" yaml:"is_cluster"`
	DbIndex     int    `json:"db_index" yaml:"db_index"`
	Description string `json:"description" yaml:"description"`
}

package factory

import "testing"

func TestParseConfig_Consul(t *testing.T) {
	cfg, err := ParseConfig("consul://127.0.0.1:8500?path=goone/config&timeout=5s")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendConsul {
		t.Fatalf("backend=%q want=%q", cfg.Backend, BackendConsul)
	}
	if cfg.Path != "goone/config" {
		t.Fatalf("path=%q", cfg.Path)
	}
	if len(cfg.Addrs) != 1 || cfg.Addrs[0] != "127.0.0.1:8500" {
		t.Fatalf("addrs=%v", cfg.Addrs)
	}
}

func TestParseConfig_Etcd(t *testing.T) {
	cfg, err := ParseConfig("etcd://127.0.0.1:2379?path=/goone/config&prefix=true")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendEtcd {
		t.Fatalf("backend=%q want=%q", cfg.Backend, BackendEtcd)
	}
	if cfg.Path != "/goone/config" || cfg.EtcdPrefix != true {
		t.Fatalf("path=%q prefix=%v", cfg.Path, cfg.EtcdPrefix)
	}
}

func TestParseConfig_K8S(t *testing.T) {
	cfg, err := ParseConfig("k8s://?namespace=default&label=app=goone")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendK8S {
		t.Fatalf("backend=%q want=%q", cfg.Backend, BackendK8S)
	}
	if cfg.KubeNamespace != "default" {
		t.Fatalf("namespace=%q", cfg.KubeNamespace)
	}
	if cfg.KubeLabelSelect != "app=goone" {
		t.Fatalf("label=%q", cfg.KubeLabelSelect)
	}
}

func TestParseConfig_Apollo(t *testing.T) {
	cfg, err := ParseConfig("apollo://?appid=goone&endpoint=http://127.0.0.1:8080&cluster=dev&namespace=application.yaml&backup=true")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendApollo {
		t.Fatalf("backend=%q want=%q", cfg.Backend, BackendApollo)
	}
	if cfg.ApolloAppID != "goone" || cfg.ApolloEndpoint != "http://127.0.0.1:8080" {
		t.Fatalf("appid/endpoint mismatch: %q %q", cfg.ApolloAppID, cfg.ApolloEndpoint)
	}
	if cfg.ApolloNamespaces != "application.yaml" || cfg.ApolloBackup != true {
		t.Fatalf("namespace/backup mismatch: %q %v", cfg.ApolloNamespaces, cfg.ApolloBackup)
	}
}

func TestParseConfig_Nacos(t *testing.T) {
	cfg, err := ParseConfig("nacos://127.0.0.1:8848?dataid=app.yaml,db.yaml&group=DEFAULT_GROUP&namespace_id=public&timeout=5s")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendNacos {
		t.Fatalf("backend=%q want=%q", cfg.Backend, BackendNacos)
	}
	if len(cfg.Addrs) != 1 || cfg.Addrs[0] != "127.0.0.1:8848" {
		t.Fatalf("addrs=%v", cfg.Addrs)
	}
	if len(cfg.NacosDataIDs) != 2 || cfg.NacosDataIDs[0] != "app.yaml" || cfg.NacosDataIDs[1] != "db.yaml" {
		t.Fatalf("dataids=%v", cfg.NacosDataIDs)
	}
	if cfg.NacosGroup != "DEFAULT_GROUP" || cfg.NacosNamespaceID != "public" {
		t.Fatalf("group/ns mismatch: %q %q", cfg.NacosGroup, cfg.NacosNamespaceID)
	}
}



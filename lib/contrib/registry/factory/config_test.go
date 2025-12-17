package factory

import "testing"

func TestParseConfig_DefaultZK_NoScheme(t *testing.T) {
	cfg, err := ParseConfig("127.0.0.1:2181")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendZooKeeper {
		t.Fatalf("backend = %q, want %q", cfg.Backend, BackendZooKeeper)
	}
	if cfg.ServiceName != "online" {
		t.Fatalf("serviceName = %q, want %q", cfg.ServiceName, "online")
	}
	if cfg.RootPath != "/" {
		t.Fatalf("rootPath = %q, want %q", cfg.RootPath, "/")
	}
	if len(cfg.Addrs) != 1 || cfg.Addrs[0] != "127.0.0.1:2181" {
		t.Fatalf("addrs = %#v, want [127.0.0.1:2181]", cfg.Addrs)
	}
}

func TestParseConfig_ZK_WithQuery(t *testing.T) {
	cfg, err := ParseConfig("zk://127.0.0.1:2181?root=/microservices&service=goone_online&timeout=5s")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendZooKeeper {
		t.Fatalf("backend = %q, want %q", cfg.Backend, BackendZooKeeper)
	}
	if cfg.RootPath != "/microservices" {
		t.Fatalf("rootPath = %q, want %q", cfg.RootPath, "/microservices")
	}
	if cfg.ServiceName != "goone_online" {
		t.Fatalf("serviceName = %q, want %q", cfg.ServiceName, "goone_online")
	}
}

func TestParseConfig_K8S_EmptyHostAllowed(t *testing.T) {
	cfg, err := ParseConfig("k8s://?service=online&incluster=true")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.Backend != BackendK8S {
		t.Fatalf("backend = %q, want %q", cfg.Backend, BackendK8S)
	}
	if cfg.ServiceName != "online" {
		t.Fatalf("serviceName = %q, want %q", cfg.ServiceName, "online")
	}
	if cfg.InCluster != true {
		t.Fatalf("inCluster = %v, want true", cfg.InCluster)
	}
}



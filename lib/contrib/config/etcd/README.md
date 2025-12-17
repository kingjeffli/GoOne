# Etcd Config

```go
import (
	cfg "github.com/Iori372552686/GoOne/lib/contrib/config/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// create a etcd client
client, err := clientv3.New(clientv3.Config{
    Endpoints:   []string{"127.0.0.1:2379"},
    DialTimeout: time.Second,
    DialOptions: []grpc.DialOption{grpc.WithBlock()},
})
if err != nil {
    log.Fatal(err)
}

// configure the source, "path" is required
source, err := cfg.New(client, cfg.WithPath("/app-config"), cfg.WithPrefix(true))
if err != nil {
    log.Fatalln(err)
}

// load kvs
kvs, err := source.Load()
_ = kvs
_ = err

```


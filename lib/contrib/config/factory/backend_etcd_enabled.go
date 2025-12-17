//go:build config_etcd
// +build config_etcd

package factory

import (
	"context"
	"fmt"

	contribconfig "github.com/Iori372552686/GoOne/lib/contrib/config"
	conf_etcd "github.com/Iori372552686/GoOne/lib/contrib/config/etcd"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdClient struct {
	contribconfig.Client
	cli *clientv3.Client
}

func (e *etcdClient) Close() error {
	if e.cli != nil {
		return e.cli.Close()
	}
	return nil
}

func newEtcdClient(cfg Config) (contribconfig.Client, error) {
	if len(cfg.Addrs) == 0 {
		return nil, fmt.Errorf("etcd: missing address")
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Addrs,
		DialTimeout: cfg.Timeout,
	})
	if err != nil {
		return nil, err
	}
	src, err := conf_etcd.New(cli,
		conf_etcd.WithContext(context.Background()),
		conf_etcd.WithPath(cfg.Path),
		conf_etcd.WithPrefix(cfg.EtcdPrefix),
	)
	if err != nil {
		_ = cli.Close()
		return nil, err
	}
	return &etcdClient{Client: contribconfig.Wrap(src, nil), cli: cli}, nil
}



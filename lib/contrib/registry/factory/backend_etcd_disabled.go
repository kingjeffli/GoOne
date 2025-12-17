//go:build !registry_etcd
// +build !registry_etcd

package factory

import (
	"fmt"

	"github.com/Iori372552686/GoOne/lib/contrib/registry"
)

func newEtcdClient(cfg Config) (registry.Client, error) {
	_ = cfg
	return nil, fmt.Errorf("etcd backend not enabled (build with: -tags registry_etcd)")
}

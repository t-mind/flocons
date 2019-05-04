package cluster

import "github.com/samuel/go-zookeeper/zk"

type ZookeeperClient interface {
	Create(path string, data []byte, flags int32, acl []zk.ACL) (string, error)
	Set(path string, data []byte, version int32) (*zk.Stat, error)
	Delete(path string, version int32) error
	Exists(path string) (bool, *zk.Stat, error)
	ExistsW(path string) (bool, *zk.Stat, <-chan zk.Event, error)
	Get(path string) ([]byte, *zk.Stat, error)
	GetW(path string) ([]byte, *zk.Stat, <-chan zk.Event, error)
	Children(path string) ([]string, *zk.Stat, error)
	ChildrenW(path string) ([]string, *zk.Stat, <-chan zk.Event, error)
	Close()
}

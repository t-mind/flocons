package mock

import "github.com/macq/flocons/cluster"

type NullTopologyClient struct{}

func (c *NullTopologyClient) Nodes() map[string]*cluster.NodeInfo {
	return make(map[string]*cluster.NodeInfo)
}
func (c *NullTopologyClient) GetNodeForObject(p string) *cluster.NodeInfo { return nil }
func (c *NullTopologyClient) Close()                                      {}

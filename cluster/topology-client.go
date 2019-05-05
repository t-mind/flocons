package cluster

import (
	"context"
	"encoding/json"
	"path"
	"sync"
	"time"

	"github.com/macq/flocons/config"
	"github.com/samuel/go-zookeeper/zk"
)

const RETRY_TIMEOUT time.Duration = 5000 * time.Millisecond

type TopologyClient interface {
	Nodes() map[string]*NodeInfo
	GetNodeForObject(p string) *NodeInfo
	Close()
}

type topologyClient struct {
	nodes           map[string]*NodeInfo
	currentNodeName string
	config          *config.Config
	zkFactory       ZookeeperClientFactory
	zkClient        ZookeeperClient
	zkEvents        <-chan zk.Event
	zkPath          string
	childrenEvent   <-chan zk.Event
	dispatcher      Dispatcher
	zkClientLock    *sync.RWMutex
	cancel          context.CancelFunc
}

type ZookeeperClientFactory func(servers []string, sessionTimeout time.Duration) (ZookeeperClient, <-chan zk.Event, error)

func NewClient(config *config.Config, dispatcher Dispatcher) TopologyClient {
	return NewClientWithZookeperClientFactory(config, func(servers []string, sessionTimeout time.Duration) (ZookeeperClient, <-chan zk.Event, error) {
		return zk.Connect(servers, time.Second)
	}, dispatcher)
}

func NewClientWithZookeperClientFactory(config *config.Config, factory ZookeeperClientFactory, dispatcher Dispatcher) TopologyClient {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	client := topologyClient{
		nodes:           make(map[string]*NodeInfo, 0),
		currentNodeName: config.Node.Name,
		config:          config,
		zkFactory:       factory,
		zkClient:        nil,
		zkEvents:        nil,
		zkPath:          path.Join("/flocons", config.Namespace, config.Node.Name),
		dispatcher:      dispatcher,
		zkClientLock:    &sync.RWMutex{},
		cancel:          cancel,
	}
	go client.connect(ctx)
	return &client
}

func (c *topologyClient) Nodes() map[string]*NodeInfo {
	return c.nodes
}

func (c *topologyClient) GetNodeForObject(p string) *NodeInfo {
	nodeName, _ := c.dispatcher.Get(p)
	if nodeName != "" {
		node, _ := c.nodes[nodeName]
		return node
	}
	return nil
}

func (c *topologyClient) connect(ctx context.Context) {
	logger.Debugf("Client for node %s start connection to zookeeper", c.currentNodeName)
	for {
		if err := c.doConnect(); err == nil {
			c.watchConnection(ctx)
		}
		c.zkClientLock.Lock()
		c.zkClient = nil
		c.zkEvents = nil
		c.zkClientLock.Unlock()
		if ctx.Err() == context.Canceled {
			logger.Infof("Client for node %s terminated", c.currentNodeName)
			return
		}
		logger.Infof("Will retry a connection to zookeeper in %d ms\n", RETRY_TIMEOUT)
		time.Sleep(RETRY_TIMEOUT)
	}
}

func (c *topologyClient) doConnect() error {
	c.zkClientLock.RLock()
	defer c.zkClientLock.RUnlock()
	zkClient, zkEvents, err := c.zkFactory(c.config.Zookeeper, time.Second)
	if err != nil {
		logger.Errorf("Could not create zookeeper connection %s\n", err)
		return err
	} else {
		logger.Debugf("Client for node %s created connection to zookeeper", c.currentNodeName)
		c.zkClient = zkClient
		c.zkEvents = zkEvents
	}
	return nil
}

func (c *topologyClient) watchConnection(ctx context.Context) {
	for {
		select {
		case event, more := <-c.zkEvents:
			if !more {
				return
			}
			if event.Type == zk.EventSession {
				switch event.State {
				case zk.StateDisconnected:
					logger.Warnf("Client for node %s disconnected from zookeeper", c.currentNodeName)
					c.clear()
					return
				case zk.StateConnected:
					logger.Infof("Client for node %s connected to zookeeper", c.currentNodeName)
					if err := c.startNodeManagement(); err != nil {
						logger.Errorf("Could not start node watching %s", err)
						c.zkClient.Close()
					}
				}
			}
		case <-ctx.Done():
			c.clear()
			return
		}
	}
}

func (c *topologyClient) startNodeManagement() error {
	var createPath func(p string) error
	createPath = func(p string) error {
		logger.Debugf("Create basic path %s", p)
		parent := path.Dir(p)
		if parent != "/" {
			if err := createPath(parent); err != nil {
				return err
			}
		}
		if _, err := c.zkClient.Create(p, nil, 0, zk.WorldACL(zk.PermAll)); err != nil && err != zk.ErrNodeExists {
			logger.Errorf("Could not create path %s: %s", p, err)
			return err
		}
		return nil
	}
	if err := createPath(path.Dir(c.zkPath)); err != nil {
		return err
	}
	if err := c.updateNodeInfo(); err != nil {
		logger.Errorf("Could not create node %s: %s", c.zkPath, err)
		return err
	}
	go c.watchNodes()
	return nil
}

func (c *topologyClient) updateNodeInfo() error {
	nodeInfo := NodeInfo{
		Name:    c.config.Node.Name,
		Address: c.config.Node.ExternalAddress,
		Shard:   c.config.Node.Shard,
	}
	js, _ := json.Marshal(nodeInfo)
	_, err := c.zkClient.Create(c.zkPath, js, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err == zk.ErrNodeExists {
		_, err = c.zkClient.Set(c.zkPath, js, 0)
	}
	return err
}

func (c *topologyClient) watchNodes() {
	for {
		channel, err := c.getNodesAndWatch()
		if err != nil {
			return
		}
		for event := range channel {
			logger.Debugf("Received event for node %s in node children of type %s for path %s", c.currentNodeName, event.Type, event.Path)
			switch event.Type {
			case zk.EventNotWatching:
				logger.Warnf("watching closed")
				return
			case zk.EventNodeChildrenChanged:
				logger.Debugf("Node children changed")
			}
		}
	}
}

func (c *topologyClient) getNodesAndWatch() (<-chan zk.Event, error) {
	c.zkClientLock.RLock()
	defer c.zkClientLock.RUnlock()

	dir := path.Dir(c.zkPath)
	if c.zkClient == nil {
		return nil, zk.ErrConnectionClosed
	}
	names, _, channel, err := c.zkClient.ChildrenW(dir)
	if err != nil {
		if err != zk.ErrClosing && err != zk.ErrConnectionClosed {
			logger.Errorf("Could not watch %s children: %s\n", dir, err)
			c.zkClient.Close()
		}
		return nil, err
	}
	c.updateNodes(names)
	return channel, nil
}

func (c *topologyClient) updateNodes(names []string) {
delLoop:
	for key, _ := range c.nodes {
		for _, name := range names {
			if name == key {
				continue delLoop
			}
		}
		c.removeNode(key)
		delete(c.nodes, key)
	}
	for _, name := range names {
		if !c.hasNode(name) {
			c.addNode(name)
		}
	}
	c.dispatcher.Set(names)
}

func (c *topologyClient) hasNode(name string) bool {
	_, ok := c.nodes[name]
	return ok
}

func (c *topologyClient) addNode(name string) {
	if name == c.currentNodeName {
		return
	}
	logger.Infof("Client for node %s has detected client for node %s", c.currentNodeName, name)
	var node NodeInfo
	data, _, _ := c.zkClient.Get(path.Join(path.Dir(c.zkPath), name))
	json.Unmarshal(data, &node)
	logger.Debugf("Client %s is in shard %s with address %s", node.Name, node.Shard, node.Address)
	c.nodes[name] = &node
}

func (c *topologyClient) removeNode(name string) {
	if name == c.currentNodeName {
		return
	}
	logger.Infof("Client for node %s has seen disconnection of client for node %s", c.currentNodeName, name)
	delete(c.nodes, name)
}

func (c *topologyClient) clear() {
	for key, _ := range c.nodes {
		delete(c.nodes, key)
	}
	c.dispatcher.Clear()
}

func (c *topologyClient) Close() {
	c.zkClientLock.RLock()
	defer c.zkClientLock.RUnlock()
	if c.zkClient != nil {
		c.zkClient.Close()
	}
	c.cancel()
}

type NodeInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Shard   string `json:"shard"`
}

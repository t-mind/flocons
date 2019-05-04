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

type TopologyClient struct {
	Nodes           map[string]*NodeInfo
	CurrentNodeName string
	config          *config.Config
	zkFactory       ZookeeperClientFactory
	zkClient        ZookeeperClient
	zkEvents        <-chan zk.Event
	zkClientLock    *sync.Mutex
	zkPath          string
	childrenEvent   <-chan zk.Event
	cancel          context.CancelFunc
}

type ZookeeperClientFactory func(servers []string, sessionTimeout time.Duration) (ZookeeperClient, <-chan zk.Event, error)

func NewClient(config *config.Config) *TopologyClient {
	return NewClientWithZookeperClientFactory(config, func(servers []string, sessionTimeout time.Duration) (ZookeeperClient, <-chan zk.Event, error) {
		return zk.Connect(servers, time.Second)
	})
}

func NewClientWithZookeperClientFactory(config *config.Config, factory ZookeeperClientFactory) *TopologyClient {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	client := TopologyClient{
		Nodes:           make(map[string]*NodeInfo, 0),
		CurrentNodeName: config.Node.Name,
		config:          config,
		zkFactory:       factory,
		zkClient:        nil,
		zkEvents:        nil,
		zkClientLock:    &sync.Mutex{},
		zkPath:          path.Join("/flocons", config.Namespace, config.Node.Name),
		cancel:          cancel,
	}
	go client.connect(ctx)
	return &client
}

func (c *TopologyClient) connect(ctx context.Context) {
	logger.Debugf("Client for node %s start connection to zookeeper", c.CurrentNodeName)
	for {
		if err := c.doConnect(); err == nil {
			c.watchConnection(ctx)
		}
		c.zkClientLock.Lock()
		c.zkClient = nil
		c.zkEvents = nil
		c.zkClientLock.Unlock()
		if ctx.Err() == context.Canceled {
			logger.Infof("Client for node %s terminated", c.CurrentNodeName)
			return
		}
		logger.Infof("Will retry a connection to zookeeper in %d ms\n", RETRY_TIMEOUT)
		time.Sleep(RETRY_TIMEOUT)
	}
}

func (c *TopologyClient) doConnect() error {
	c.zkClientLock.Lock()
	defer c.zkClientLock.Unlock()
	zkClient, zkEvents, err := c.zkFactory(c.config.Zookeeper, time.Second)
	if err != nil {
		logger.Errorf("Could not create zookeeper connection %s\n", err)
		return err
	} else {
		logger.Debugf("Client for node %s created connection to zookeeper", c.CurrentNodeName)
		c.zkClient = zkClient
		c.zkEvents = zkEvents
	}
	return nil
}

func (c *TopologyClient) watchConnection(ctx context.Context) {
	for {
		select {
		case event, more := <-c.zkEvents:
			if !more {
				return
			}
			if event.Type == zk.EventSession {
				switch event.State {
				case zk.StateDisconnected:
					logger.Warnf("Client for node %s disconnected from zookeeper", c.CurrentNodeName)
					c.clear()
					return
				case zk.StateConnected:
					logger.Infof("Client for node %s connected to zookeeper", c.CurrentNodeName)
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

func (c *TopologyClient) startNodeManagement() error {
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

func (c *TopologyClient) updateNodeInfo() error {
	nodeInfo := NodeInfo{
		Name:    c.config.Node.Name,
		Address: c.config.Node.ExternalAddress,
		Shard:   c.config.Node.Shard,
	}
	js, jserr := json.Marshal(nodeInfo)
	if jserr != nil {
		logger.Fatalf("COULD NOT MARSHALL JSON %s", jserr)
	}
	logger.Infof("CREATE NODE %s WITH JS %s", c.zkPath, js)
	_, err := c.zkClient.Create(c.zkPath, js, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err == zk.ErrNodeExists {
		_, err = c.zkClient.Set(c.zkPath, js, 0)
	}
	return err
}

func (c *TopologyClient) watchNodes() {
	for {
		channel, err := c.getNodesAndWatch()
		if err != nil {
			return
		}
		for event := range channel {
			logger.Debugf("Received event for node %s in node children of type %s for path %s", c.CurrentNodeName, event.Type, event.Path)
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

func (c *TopologyClient) getNodesAndWatch() (<-chan zk.Event, error) {
	c.zkClientLock.Lock()
	defer c.zkClientLock.Unlock()

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

func (c *TopologyClient) updateNodes(names []string) {
delLoop:
	for key, _ := range c.Nodes {
		for _, name := range names {
			if name == key {
				continue delLoop
			}
		}
		c.removeNode(key)
		delete(c.Nodes, key)
	}
	for _, name := range names {
		if !c.hasNode(name) {
			c.addNode(name)
		}
	}
}

func (c *TopologyClient) hasNode(name string) bool {
	_, ok := c.Nodes[name]
	return ok
}

func (c *TopologyClient) addNode(name string) {
	if name == c.CurrentNodeName {
		return
	}
	logger.Infof("Client for node %s has detected client for node %s", c.CurrentNodeName, name)
	var node NodeInfo
	data, _, _ := c.zkClient.Get(path.Join(path.Dir(c.zkPath), name))
	json.Unmarshal(data, &node)
	logger.Debugf("Client %s is in shard %s with address %s", node.Name, node.Shard, node.Address)
	c.Nodes[name] = &node
}

func (c *TopologyClient) removeNode(name string) {
	if name == c.CurrentNodeName {
		return
	}
	logger.Infof("Client for node %s has seen disconnection of client for node %s", c.CurrentNodeName, name)
	delete(c.Nodes, name)
}

func (c *TopologyClient) clear() {
	for key, _ := range c.Nodes {
		delete(c.Nodes, key)
	}
}

func (c *TopologyClient) Close() {
	c.zkClientLock.Lock()
	defer c.zkClientLock.Unlock()
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

package mock

import (
	"path"
	"time"

	"github.com/macq/flocons/cluster"
	"github.com/samuel/go-zookeeper/zk"
)

type Zookeeper struct {
	root   *ZookeeperNode
	Events chan zk.Event
}

type ZookeeperClient struct {
	zk     *Zookeeper
	events chan zk.Event
	closed bool
}

func NewZookeeper() *Zookeeper {
	return &Zookeeper{
		root:   newZookeeperNode(nil, nil),
		Events: make(chan zk.Event, 1000),
	}
}

func (z *Zookeeper) GetFactory() cluster.ZookeeperClientFactory {
	return func(servers []string, sessionTimeout time.Duration) (cluster.ZookeeperClient, <-chan zk.Event, error) {
		client := ZookeeperClient{
			zk:     z,
			events: make(chan zk.Event, 1000),
		}
		z.Events <- zk.Event{Type: zk.EventSession, State: zk.StateConnected}
		client.events <- zk.Event{Type: zk.EventSession, State: zk.StateConnected}
		return &client, client.events, nil
	}
}

func (c *ZookeeperClient) Create(p string, data []byte, flags int32, acl []zk.ACL) (string, error) {
	if c.closed {
		return "", zk.ErrConnectionClosed
	}
	if acl == nil {
		return "", zk.ErrInvalidACL
	}
	dir := path.Dir(p)
	name := path.Base(p)
	parent, err := c.zk.getNode(dir)
	if err != nil {
		return "", err
	}
	var ephemeralNode *ZookeeperClient
	if flags&zk.FlagEphemeral == zk.FlagEphemeral {
		ephemeralNode = c
	}
	err = parent.addChild(name, data, ephemeralNode)
	if err != nil {
		return "", err
	}
	c.zk.Events <- zk.Event{Type: zk.EventNodeCreated, Path: p}
	return "", nil
}

func (c *ZookeeperClient) Set(p string, data []byte, version int32) (*zk.Stat, error) {
	if c.closed {
		return nil, zk.ErrConnectionClosed
	}
	node, err := c.zk.getNode(p)
	if err != nil {
		return nil, err
	}
	node.setData(data)

	c.zk.Events <- zk.Event{Type: zk.EventNodeDataChanged, Path: p}
	return nil, nil
}

func (c *ZookeeperClient) Delete(p string, version int32) error {
	if c.closed {
		return zk.ErrConnectionClosed
	}
	dir := path.Dir(p)
	name := path.Base(p)
	parent, err := c.zk.getNode(dir)
	if err != nil {
		return err
	}
	err = parent.removeChild(name)
	if err != nil {
		return err
	}
	c.zk.Events <- zk.Event{Type: zk.EventNodeDeleted, Path: p}
	return nil
}

func (c *ZookeeperClient) Exists(p string) (bool, *zk.Stat, error) {
	if c.closed {
		return false, nil, zk.ErrConnectionClosed
	}
	parent, err := c.zk.getNode(path.Dir(p))
	if err != nil {
		return false, nil, err
	}
	return parent.hasChild(path.Base(p)), nil, nil
}

func (c *ZookeeperClient) ExistsW(p string) (bool, *zk.Stat, <-chan zk.Event, error) {
	exists, stat, err := c.Exists(p)
	return exists, stat, nil, err
}

func (c *ZookeeperClient) Get(p string) ([]byte, *zk.Stat, error) {
	if c.closed {
		return nil, nil, zk.ErrConnectionClosed
	}
	node, err := c.zk.getNode(p)
	if err != nil {
		return nil, nil, err
	}
	return node.data, nil, nil
}

func (c *ZookeeperClient) GetW(p string) ([]byte, *zk.Stat, <-chan zk.Event, error) {
	if c.closed {
		return nil, nil, nil, zk.ErrConnectionClosed
	}
	node, err := c.zk.getNode(p)
	if err != nil {
		return nil, nil, nil, err
	}
	watcher := ZookeeperNodeWatcher{
		client: c,
		events: make(chan zk.Event),
	}
	node.dataWatchers = append(node.dataWatchers, watcher)
	return node.data, nil, watcher.events, nil
}

func (c *ZookeeperClient) Children(p string) ([]string, *zk.Stat, error) {
	if c.closed {
		return nil, nil, zk.ErrConnectionClosed
	}
	node, err := c.zk.getNode(p)
	if err != nil {
		return nil, nil, err
	}
	return node.getChildren(), nil, nil
}

func (c *ZookeeperClient) ChildrenW(p string) ([]string, *zk.Stat, <-chan zk.Event, error) {
	if c.closed {
		return nil, nil, nil, zk.ErrConnectionClosed
	}
	node, err := c.zk.getNode(p)
	if err != nil {
		return nil, nil, nil, err
	}
	watcher := ZookeeperNodeWatcher{
		client: c,
		events: make(chan zk.Event, 1),
	}
	node.childrenWatchers = append(node.childrenWatchers, watcher)
	return node.getChildren(), nil, watcher.events, err
}

func (c *ZookeeperClient) Close() {
	if c.closed {
		panic("Can't close a client twice")
	}
	c.closed = true
	c.zk.root.clearWatchers(c)
	c.zk.root.clear(c)
	c.events <- zk.Event{Type: zk.EventSession, State: zk.StateDisconnected}
	close(c.events)
	c.zk.Events <- zk.Event{Type: zk.EventSession, State: zk.StateDisconnected}
}

func (z *Zookeeper) Clear() {
	z.root.clear(nil)
}

func (z *Zookeeper) getNode(p string) (*ZookeeperNode, error) {
	parent := path.Dir(p)
	if parent == "/" {
		return z.root, nil
	}
	parentNode, err := z.getNode(parent)
	if err != nil {
		return nil, err
	}
	return parentNode.getChild(path.Base(p))
}

type ZookeeperNode struct {
	data                []byte
	children            map[string]*ZookeeperNode
	childrenWatchers    []ZookeeperNodeWatcher
	dataWatchers        []ZookeeperNodeWatcher
	ephemeralNodeClient *ZookeeperClient
}

func newZookeeperNode(data []byte, ephemeralNodeClient *ZookeeperClient) *ZookeeperNode {
	return &ZookeeperNode{
		data:                data,
		children:            make(map[string]*ZookeeperNode),
		childrenWatchers:    make([]ZookeeperNodeWatcher, 0),
		dataWatchers:        make([]ZookeeperNodeWatcher, 0),
		ephemeralNodeClient: ephemeralNodeClient,
	}
}

func (n *ZookeeperNode) setData(data []byte) {
	n.data = data
	n.sendDataEvent(zk.EventNodeDataChanged)
}

func (n *ZookeeperNode) hasChild(name string) bool {
	_, ok := n.children[name]
	return ok
}

func (n *ZookeeperNode) addChild(name string, data []byte, ephemeralNodeClient *ZookeeperClient) error {
	if n.hasChild(name) {
		return zk.ErrNodeExists
	}
	n.children[name] = newZookeeperNode(data, ephemeralNodeClient)
	n.sendChildrenEvent(zk.EventNodeChildrenChanged)
	return nil
}

func (n *ZookeeperNode) removeChild(name string) error {
	if !n.hasChild(name) {
		return zk.ErrNoNode
	}
	node, _ := n.children[name]
	if len(node.children) > 0 {
		return zk.ErrNotEmpty
	}
	for _, watcher := range append(node.childrenWatchers, node.dataWatchers...) {
		close(watcher.events)
	}
	delete(n.children, name)
	n.sendChildrenEvent(zk.EventNodeChildrenChanged)
	return nil
}

func (n *ZookeeperNode) getChild(name string) (*ZookeeperNode, error) {
	if node, ok := n.children[name]; ok {
		return node, nil
	}
	return nil, zk.ErrNoNode
}

func (n *ZookeeperNode) getChildren() []string {
	names := make([]string, 0, len(n.children))
	for name, _ := range n.children {
		names = append(names, name)
	}
	return names
}

func (n *ZookeeperNode) sendChildrenEvent(childrenEventType zk.EventType) {
	childrenWatchers := n.childrenWatchers
	n.childrenWatchers = nil
	for _, watcher := range childrenWatchers {
		watcher.events <- zk.Event{Type: childrenEventType}
		close(watcher.events)
	}
}
func (n *ZookeeperNode) sendDataEvent(dataEventType zk.EventType) {
	dataWatchers := n.dataWatchers
	n.dataWatchers = nil
	for _, watcher := range dataWatchers {
		watcher.events <- zk.Event{Type: dataEventType}
		close(watcher.events)
	}
}

func (n *ZookeeperNode) clear(ephemeralNodeClient *ZookeeperClient) {
	for name, node := range n.children {
		node.clear(ephemeralNodeClient)
		if ephemeralNodeClient == nil || node.ephemeralNodeClient == ephemeralNodeClient {
			n.removeChild(name)
		}
	}
}

func (n *ZookeeperNode) clearWatchers(client *ZookeeperClient) {
	filterWatchers := func(watchers []ZookeeperNodeWatcher) []ZookeeperNodeWatcher {
		result := make([]ZookeeperNodeWatcher, 0, len(watchers))
		for _, watcher := range watchers {
			if watcher.client == client {
				close(watcher.events)
			} else {
				result = append(result, watcher)
			}
		}
		return result
	}
	for _, node := range n.children {
		node.clearWatchers(client)
		node.childrenWatchers = filterWatchers(node.childrenWatchers)
		node.dataWatchers = filterWatchers(node.dataWatchers)
	}
}

type ZookeeperNodeWatcher struct {
	events chan zk.Event
	client *ZookeeperClient
}

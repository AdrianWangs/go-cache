package cache

import (
	"fmt"
	"sync"

	"github.com/AdrianWangs/go-cache/internal/interfaces"
	"github.com/AdrianWangs/go-cache/internal/peers"
	"github.com/AdrianWangs/go-cache/internal/singleflight"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
)

// 一个Group可以认为是一个缓存的命名空间
type Group struct {
	name      string            // 缓存命名空间
	getter    interfaces.Getter // 缓存未命中时获取源数据的回调
	mainCache cache             // 并发缓存
	peers     peers.PeerPicker  // 节点选择器

	loader *singleflight.Group // 用于管理不同key的请求(call),同一时间进来的请求不需要重复执行
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 创建一个新的Group, 使用者需要提供回调函数, 当缓存未命中时, 调用回调函数获取源数据
//
// 传入参数:
//   - name: 缓存命名空间
//   - cacheBytes: 缓存大小
//   - getter: 缓存未命中时获取源数据的回调
//
// 返回值:
//   - *Group: 返回一个Group
func NewGroup(name string, cacheBytes int64, getter interfaces.Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	logger.Infof("Create cache group: %s, size: %d bytes", name, cacheBytes)
	return g
}

// GetGroup 获取一个Group
//
// 传入参数:
//   - name: 缓存命名空间
//
// 返回值:
//   - *Group: 返回一个Group
func GetGroup(name string) *Group {

	// 使用读锁, 因为获取Group时, 不会修改Group，所以不会发生冲突
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 获取缓存
// 如果缓存未命中, 则调用回调函数获取源数据
//
// 传入参数:
//   - key: 缓存的key
//
// 返回值:
//   - ByteView: 缓存的value
//   - error: 错误信息
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		logger.Debugf("[Cache] hit for key: %s", key)
		return v, nil
	}

	return g.load(key)
}

// load 加载缓存,只有在缓存不存在的时候才会执行这个操作
//
// 传入参数:
//   - key: 缓存的key
//
// 返回值:
//   - ByteView: 缓存的value
//   - error: 错误信息
func (g *Group) load(key string) (value ByteView, err error) {
	logger.Info("[load] 缓存未命中，开始加载缓存")

	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 尝试从远程节点获取数据
		if g.peers != nil {
			// 选择一个节点，这个节点负责这个key的缓存
			if peer, ok := g.peers.PickPeer(key); ok {
				// TODO 这里可以选择使用http协议或者protobuf协议
				// if value, err = g.getFromPeer(peer, key); err == nil {
				// 	return value, nil
				// }
				if value, err = g.getFromPeerByProto(peer, key); err == nil {
					return value, nil
				}
				logger.Errorf("[GoCache] Failed to get from peer: %v", err)
			}
		}
		// 如果远程节点获取数据失败, 则从本地获取数据
		return g.getLocally(key)
	})
	if err != nil {
		return ByteView{}, err
	}
	return viewi.(ByteView), nil
}

// getLocally 调用回调函数获取源数据
//
// 传入参数:
//   - key: 缓存的key
//
// 返回值:
//   - ByteView: 缓存的value
//   - error: 错误信息
func (g *Group) getLocally(key string) (ByteView, error) {
	// 通过回调函数获取源数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		logger.Errorf("[getLocally] Failed to get locally: %v", err)
		return ByteView{}, err
	}

	value := ByteView{bytes: cloneBytes(bytes)}

	// 将源数据添加到缓存
	g.populateCache(key, value)
	logger.Debugf("[getLocally] Got data locally for key: %s, size: %d bytes", key, len(bytes))

	return value, nil
}

// getFromPeer 调用对应的远端peer获取缓存
//
// 传入参数:
//   - peer: 远端peer
//   - key: 缓存的key
//
// 返回值:
//   - ByteView: 缓存的value
//   - error: 错误信息
func (g *Group) getFromPeer(peer peers.PeerGetter, key string) (ByteView, error) {
	logger.Debugf("[getFromPeer] Get %s/%s from peer", g.name, key)
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	logger.Debugf("[getFromPeer] Successfully got data from peer for key: %s, size: %d bytes", key, len(bytes))
	return ByteView{bytes: bytes}, nil
}

// getFromPeerByProto 调用对应的远端peer获取缓存,使用protobuf协议
//
// 传入参数:
//   - peer: 远端peer
//   - key: 缓存的key
//
// 返回值:
//   - ByteView: 缓存的value
//   - error: 错误信息
func (g *Group) getFromPeerByProto(peer peers.PeerGetter, key string) (ByteView, error) {
	logger.Debugf("[getFromPeerByProto] Get %s/%s from peer", g.name, key)

	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}

	resp := &pb.Response{}

	if err := peer.GetByProto(req, resp); err != nil {
		logger.Errorf("[getFromPeerByProto] Failed to get from peer: %v", err)
		return ByteView{}, err
	}

	return ByteView{bytes: resp.Value}, nil
}

// populateCache 将源数据添加到缓存
//
// 传入参数:
//   - key: 缓存的key
//   - value: 缓存的value
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
	logger.Debugf("[populateCache] Added key %s to cache", key)
}

// RegisterPeers 用于注册
//
// 传入参数:
//   - peers: 节点选择器
func (g *Group) RegisterPeers(peers peers.PeerPicker) {
	if g.peers != nil {
		logger.Warn("RegisterPeers has already been called")
		return
	}

	g.peers = peers
	logger.Infof("RegisterPeers for group: %s", g.name)
}

package cache

import (
	"context"
	"sync"
	"time"

	"github.com/AdrianWangs/go-cache/internal/peers"
	"github.com/AdrianWangs/go-cache/internal/singleflight"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
)

// Group is a cache namespace
type Group struct {
	name      string              // name of the cache namespace
	getter    Getter              // the getter interface used when cache miss
	mainCache *Cache              // main cache
	peers     peers.PeerPicker    // peer picker interface
	loader    *singleflight.Group // singleflight prevents redundant loads
	ttl       time.Duration       // ttl of the cache
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup creates a new Group
func NewGroup(name string, cacheBytes int64, getter Getter, ttl time.Duration) *Group {
	if getter == nil {
		logger.Fatal("nil Getter provided to NewGroup")
	}

	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: newCache(cacheBytes),
		loader:    &singleflight.Group{},
		ttl:       ttl,
	}

	groups[name] = g
	logger.Infof("Created cache group: %s, size: %d bytes", name, cacheBytes)
	return g
}

// GetGroup returns the named group previously created with NewGroup
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get retrieves a key's value from the cache, loading it from the getter if needed
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, ErrEmptyKey
	}

	// Try local cache first
	if v, ok := g.mainCache.get(key); ok {
		logger.Debugf("[Cache] hit for group:%s key:%s", g.name, key)
		return v, nil
	}

	// Cache miss, load from remote or locally
	return g.load(key)
}

// GetWithContext retrieves a key's value with context
func (g *Group) GetWithContext(ctx context.Context, key string) (ByteView, error) {
	// Basic implementation - can be extended to use context for timeouts, etc.
	return g.Get(key)
}

// Clear clears the group's cache
func (g *Group) Clear() {
	g.mainCache.clear()
	logger.Infof("Cleared cache for group: %s", g.name)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers peers.PeerPicker) {
	if g.peers != nil {
		logger.Warn("RegisterPeers called more than once")
		return
	}
	g.peers = peers
	logger.Infof("RegisterPeers for group: %s", g.name)
}

// load loads key from remote peer or locally
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// Try to get from peer first
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				// Use protobuf for communication
				value, err := g.getFromPeerWithProto(peer, key)
				if err == nil {
					logger.Debugf("[Cache] got value from peer for group:%s key:%s", g.name, key)
					return value, nil
				}
				logger.Errorf("[Cache] failed to get from peer: %v", err)
			}
		}

		// Fall back to local data source
		return g.getLocally(key)
	})

	if err != nil {
		return ByteView{}, err
	}

	return viewi.(ByteView), nil
}

// getLocally loads key by calling the getter and stores it in the cache
func (g *Group) getLocally(key string) (value ByteView, err error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		logger.Errorf("[Cache] failed to get locally: %v", err)
		return ByteView{}, WrapError(ErrTypeInternalError, "getter error", err)
	}

	// 如果bytes为nil或长度为0，认为是key不存在
	if bytes == nil || len(bytes) == 0 {
		logger.Warnf("[Cache] key not found: %s", key)
		return ByteView{}, ErrNotFound
	}

	value = ByteView{bytes: cloneBytes(bytes)}
	g.populateCache(key, value, g.ttl)
	return value, nil
}

// populateCache adds a value to the cache
func (g *Group) populateCache(key string, value ByteView, ttl time.Duration) {
	g.mainCache.add(key, value, ttl)
	logger.Debugf("[Cache] cached key:%s in group:%s", key, g.name)
}

// getFromPeerWithProto gets a value from a peer using protobuf
func (g *Group) getFromPeerWithProto(peer peers.PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}

	res := &pb.Response{}

	err := peer.GetByProto(req, res)
	if err != nil {
		return ByteView{}, err
	}

	return ByteView{bytes: res.Value}, nil
}

// GetGroups returns all registered cache groups
func GetGroups() map[string]*Group {
	mu.RLock()
	defer mu.RUnlock()

	// 创建一个副本
	result := make(map[string]*Group, len(groups))
	for k, v := range groups {
		result[k] = v
	}

	return result
}

// Stats returns statistics for this cache group
func (g *Group) Stats() CacheStats {
	return g.mainCache.stats
}

package cache

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdrianWangs/go-cache/pkg/lru"
)

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits int64 // 缓存命中次数
	Gets int64 // 缓存获取请求总数
}

// Cache is a concurrency-safe wrapper around an LRU cache
type Cache struct {
	mutex      sync.RWMutex
	lru        *lru.Cache
	cacheBytes int64
	stats      CacheStats // 缓存统计信息
}

// newCache creates a new cache with size limit
func newCache(cacheBytes int64) *Cache {
	return &Cache{
		cacheBytes: cacheBytes,
	}
}

// add adds a value to the cache
func (c *Cache) add(key string, value ByteView, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Lazy initialization
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value, ttl)
}

// get looks up a key's value from the cache
func (c *Cache) get(key string) (value ByteView, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// 增加获取计数
	atomic.AddInt64(&c.stats.Gets, 1)

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		// 增加命中计数
		atomic.AddInt64(&c.stats.Hits, 1)
		return v.(ByteView), true
	}
	return
}

// clear empties the cache
func (c *Cache) clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.lru != nil {
		c.lru.Clear()
	}
}

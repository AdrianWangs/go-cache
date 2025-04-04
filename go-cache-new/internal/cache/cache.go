package cache

import (
	"sync"

	"github.com/AdrianWangs/go-cache-new/pkg/lru"
)

// Cache is a concurrency-safe wrapper around an LRU cache
type Cache struct {
	mutex      sync.RWMutex
	lru        *lru.Cache
	cacheBytes int64
}

// newCache creates a new cache with size limit
func newCache(cacheBytes int64) *Cache {
	return &Cache{
		cacheBytes: cacheBytes,
	}
}

// add adds a value to the cache
func (c *Cache) add(key string, value ByteView) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Lazy initialization
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// get looks up a key's value from the cache
func (c *Cache) get(key string) (value ByteView, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
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

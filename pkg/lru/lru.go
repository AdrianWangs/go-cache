// Package lru provides a generic LRU cache implementation
package lru

import (
	"container/list"
	"math"
	"sync"
	"time"

	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// Value is the interface that all values stored in the cache must implement
type Value interface {
	// Len returns the size of the value in bytes
	Len() int
}

// Cache is a thread-safe LRU (Least Recently Used) cache implementation
type Cache struct {
	mutex     sync.RWMutex
	maxBytes  int64                    // maximum memory limit (0 means no limit)
	nbytes    int64                    // current memory usage in bytes
	ll        *list.List               // doubly linked list for LRU order tracking
	cache     map[string]*list.Element // hashmap for O(1) lookups
	OnEvicted func(key string, value Value)
}

// entry represents a key-value pair stored in the cache
type entry struct {
	key   string
	value Value
	exp   time.Time
}

// New creates a new LRU cache with the specified memory limit and eviction callback
func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get retrieves a value from the cache, moving it to the front (most recently used)
func (c *Cache) Get(key string) (value Value, ok bool) {
	c.mutex.RLock()
	if ele, ok := c.cache[key]; ok {
		c.mutex.RUnlock()
		// Lock for write to modify the list
		c.mutex.Lock()
		defer c.mutex.Unlock()

		// 获取条目并检查过期时间
		kv := ele.Value.(*entry)
		now := time.Now()

		// 过期就删除
		if kv.exp.Before(now) {
			logger.Infof("缓存项已过期: key=%s, 过期时间=%v, 当前时间=%v, 过期差=%v",
				key, kv.exp.Format(time.RFC3339), now.Format(time.RFC3339), now.Sub(kv.exp))
			c.ll.Remove(ele)
			delete(c.cache, key)
			c.nbytes -= int64(len(key)) + int64(kv.value.Len())
			return nil, false
		}

		// 输出剩余过期时间
		remaining := kv.exp.Sub(now)
		logger.Debugf("缓存命中: key=%s, 剩余有效时间=%v", key, remaining)

		c.ll.MoveToBack(ele)
		return kv.value, true
	}
	c.mutex.RUnlock()
	return nil, false
}

// Add adds a value to the cache, replacing an existing value if the key exists
func (c *Cache) Add(key string, value Value, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ele, ok := c.cache[key]; ok {
		// Update existing entry
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value

		// 更新过期时间
		var exp time.Time
		if ttl > 0 {
			exp = time.Now().Add(ttl)
			logger.Debugf("更新缓存项过期时间: key=%s, TTL=%v, 过期时间=%v",
				key, ttl, exp.Format(time.RFC3339))
		} else {
			// 如果ttl为0，则设置为time的max
			exp = time.Unix(math.MaxInt64, 0)
			logger.Debugf("更新缓存项永不过期: key=%s", key)
		}
		kv.exp = exp
	} else {
		// Add new entry
		var exp time.Time
		if ttl > 0 {
			exp = time.Now().Add(ttl)
			logger.Debugf("添加新缓存项: key=%s, TTL=%v, 过期时间=%v",
				key, ttl, exp.Format(time.RFC3339))
		} else {
			// 如果ttl为0，则设置为time的max
			exp = time.Unix(math.MaxInt64, 0)
			logger.Debugf("添加永不过期的缓存项: key=%s", key)
		}
		ele := c.ll.PushBack(&entry{key, value, exp})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	// Evict oldest entries if memory limit exceeded
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.removeOldest()
	}
}

// Len returns the number of items in the cache
func (c *Cache) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.ll.Len()
}

// removeOldest removes the oldest (least recently used) item from the cache
func (c *Cache) removeOldest() {
	element := c.ll.Front()
	if element != nil {
		c.ll.Remove(element)
		kv := element.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())

		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Clear empties the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.ll = list.New()
	c.cache = make(map[string]*list.Element)
	c.nbytes = 0
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ele, ok := c.cache[key]; ok {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, key)
		c.nbytes -= int64(len(key)) + int64(kv.value.Len())

		if c.OnEvicted != nil {
			c.OnEvicted(key, kv.value)
		}
		return true
	}
	return false
}

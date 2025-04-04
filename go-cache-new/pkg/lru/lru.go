// Package lru provides a generic LRU cache implementation
package lru

import (
	"container/list"
	"sync"
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
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	c.mutex.RUnlock()
	return nil, false
}

// Add adds a value to the cache, replacing an existing value if the key exists
func (c *Cache) Add(key string, value Value) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ele, ok := c.cache[key]; ok {
		// Update existing entry
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// Add new entry
		ele := c.ll.PushBack(&entry{key, value})
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

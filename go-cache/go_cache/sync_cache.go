package go_cache

import (
	"sync"

	"github.com/AdrianWangs/go-cache/lru"
)

type cache struct {
	mutex      sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

// add 添加缓存，简单对lru进行封装，确保线程安全
//
// 传入参数:
//   - key: 缓存的key
//   - value: 缓存的value
func (c *cache) add(key string, value ByteView) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// get 获取缓存，简单对lru进行封装，确保线程安全
//
// 传入参数:
//   - key: 缓存的key
//
// 返回值:
//   - value: 缓存的value
//   - ok: 是否存在
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), true
	}
	return
}

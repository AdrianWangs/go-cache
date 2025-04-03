package cache

// 双向链表
import "container/list"

// 用户自定义数据的类型
type Value interface {
	Len() int
}

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes int64                    // 最大内存
	nbytes   int64                    // 当前内存
	ll       *list.List               // 双向链表
	cache    map[string]*list.Element // 缓存
	// 可选，当内存不足时，删除缓存后，执行此方法
	OnEvicted func(key string, value Value)
}

// 双向链表的元素
type entry struct {
	key   string
	value Value
}

// New 是一个创建缓存的函数
//
// 传入参数:
//   - maxBytes: 最大内存
//   - onEvicted: 可选，当内存不足时，执行此方法，删除key,具体如何删除由用户决定
//
// 返回值:
//   - 缓存
func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 获取缓存
//
// 传入参数:
//   - key: 缓存的key
//
// 返回值:
//   - value: 缓存的value
//   - ok: 是否找到缓存
func (c *Cache) Get(key string) (value Value, ok bool) {
	// 如果找到缓存，就将它移动到队尾
	// 这里移动到队尾，表示这个缓存最近被访问了，那我们就尽量不删除它
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return nil, false
}

// Len 获取缓存中的元素个数
//
// 返回值:
//   - 缓存中的元素个数
func (c *Cache) Len() int {
	return c.ll.Len()
}

// Add 添加缓存
//
// 传入参数:
//   - key: 缓存的key
//   - value: 缓存的value
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushBack(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 如果内存超过了最大内存，则删除最老的元素
	// 如果设置为0，则不限制内存
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// RemoveOldest 是缓存淘汰策略，删除最老的元素
func (c *Cache) RemoveOldest() {
	element := c.ll.Front()
	if element != nil {
		c.ll.Remove(element)
		kv := element.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())

		// 如果用户定义了删除缓存后的回调函数，则执行
		// 具体内容由用户决定，比如可以再删除后进行日志记录，或者持久化到磁盘以方便恢复
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

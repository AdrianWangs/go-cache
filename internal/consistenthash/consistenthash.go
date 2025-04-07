// Package consistenthash implements the consistent hashing algorithm
package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"

	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map is a thread-safe implementation of a consistent hash map
type Map struct {
	mutex    sync.RWMutex
	hash     Hash           // hash function
	replicas int            // number of virtual nodes per real node
	keys     []int          // sorted hash keys
	hashMap  map[int]string // hash key -> real node mapping
}

// New creates a Map instance with the given replicas count and hash function
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 用于往一致性哈希环中添加节点
//
// 传入参数:
//   - keys: 节点名称的列表
func (m *Map) Add(keys ...string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, key := range keys {
		// Create 'replicas' virtual nodes for each real node
		for i := 0; i < m.replicas; i++ {
			// Calculate hash for virtual node
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest node in the hash to the provided key
func (m *Map) Get(key string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.keys) == 0 {
		return ""
	}

	// Calculate hash for the key
	hash := int(m.hash([]byte(key)))

	// Binary search for the first hash >= hash
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// If we reached the end, wrap around to the first replica
	if idx == len(m.keys) {
		idx = 0
	}

	node := m.hashMap[m.keys[idx]]
	logger.Debugf("一致性哈希: key=%s, hash=%d, 选中节点=%s", key, hash, node)
	return node
}

// Remove removes a node from the hash
func (m *Map) Remove(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Create a new keys slice and hashMap
	newKeys := make([]int, 0, len(m.keys)-m.replicas)
	newHashMap := make(map[int]string, len(m.hashMap)-m.replicas)

	// Copy over entries not related to the removed key
	for hash, k := range m.hashMap {
		if k != key {
			newKeys = append(newKeys, hash)
			newHashMap[hash] = k
		}
	}

	// Sort the new keys
	sort.Ints(newKeys)

	// Update the map
	m.keys = newKeys
	m.hashMap = newHashMap
}

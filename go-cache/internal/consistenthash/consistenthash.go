package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

// Map 用来存储所有hash值对应的节点
type Map struct {
	hash     Hash           //选择的hash算法
	replicas int            //虚拟节点倍数，也就是一个真实节点对应多少个虚拟节点
	keys     []int          //所有虚拟节点的hash值
	hashMap  map[int]string //虚拟节点和真实节点的映射表
}

// New 创建一个Map
//
// 传入参数:
//   - replicas 虚拟节点倍数
//   - fn 选择的hash算法
//
// 返回值:
//   - *Map 返回一个Map
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	// 默认使用crc32.ChecksumIEEE算法
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 添加节点
//
// 传入参数:
//   - keys 节点
func (m *Map) Add(keys ...string) {
	// 为每个节点添加虚拟节点，虚拟节点是根据hash算法计算出来的
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	// 对所有虚拟节点的hash值进行排序,排序是为了方便从数据所在的数据顺时针找到最近的节点
	sort.Ints(m.keys)
}

// Get 获取节点
//
// 传入参数:
//   - key 数据
//
// 返回值:
//   - string 最近节点
func (m *Map) Get(key string) string {
	// 计算key的hash值
	hash := int(m.hash([]byte(key)))
	// 顺时针找到第一个大于等于hash的节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	// 如果idx等于节点数量，说明没有找到大于等于hash的节点，则返回第一个节点
	if idx == len(m.keys) {
		idx = 0
	}
	// 返回节点
	return m.hashMap[m.keys[idx]]
}

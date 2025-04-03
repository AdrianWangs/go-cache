package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/AdrianWangs/go-cache/consistenthash"
	"github.com/AdrianWangs/go-cache/go_cache"
	"github.com/AdrianWangs/go-cache/peers"
	"github.com/sirupsen/logrus"
)

// 默认的basePath
const defaultBasePath = "/_gocache/"
const defaultReplicas = 3

// HTTPPool 实现了 PeerPicker 接口, 所以它是一个HTTP服务器
type HTTPPool struct {
	self        string                 // 自己的地址, 包括主机名/IP 和端口,比如: "localhost:8080"
	basePath    string                 // 节点间通讯地址的前缀, 默认是 /_goache/，这个前缀用来提供节点间通讯地址和提供节点服务
	mu          sync.RWMutex           // 互斥锁，确保节点选择器的安全
	peers       *consistenthash.Map    // 节点选择器
	httpGetters map[string]*httpGetter // 映射远程节点与对应的httpGetter, 键是远程节点的http地址,比如: "http://10.0.0.2:8000"
}

// 创建一个HTTPPool
//
// 传入参数:
//   - self: 自己的地址, 包括主机名/IP 和端口,比如: "localhost:8080"
//
// 返回值:
//   - *HTTPPool: 一个HTTPPool实例
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 打印日志
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理HTTP请求
//
// 传入参数:
//   - w: http.ResponseWriter
//   - r: http.Request
//
// 返回值:
//   - 无
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	logrus.Infof("[ServeHTTP] ServeHTTP %s %s", r.Method, r.URL.Path)

	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		// 如果请求路径不是以 basePath 开头, 返回错误
		http.Error(w, "HTTPPool serving unexpected path: "+r.URL.Path, http.StatusBadRequest)
		return
	}

	// 打印日志
	p.Log("%s %s", r.Method, r.URL.Path)

	// 获取请求路径,一般格式为 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		// 如果请求路径格式不正确, 返回错误
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 尝试获取group
	group := go_cache.GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 从group中获取缓存值
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 将缓存值写入响应
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(view.ByteSlice())
}

// Set 设置节点
//
// 传入参数:
//   - peers: 节点列表
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 创建peers,peers可以理解为一致性哈希的节点
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	// 添加httpGetter
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{
			baseURL: peer + p.basePath,
		}
	}
}

// PickPeer 选择一个节点
//
// 传入参数:
//   - key: 键
//
// 返回值:
//   - 节点, 是否成功
func (p *HTTPPool) PickPeer(key string) (peers.PeerGetter, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// 找到负责这个key的节点
	peer := p.peers.Get(key)
	// 如果没有peer != p.self，会导致请求自己，死循环，永远无法到达加载本地缓存那一步
	if peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}

	return nil, false
}

// 确保HTTPPool实现了peers.PeerPicker接口
var _ peers.PeerPicker = (*HTTPPool)(nil)

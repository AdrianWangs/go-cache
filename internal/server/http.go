package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/internal/consistenthash"
	"github.com/AdrianWangs/go-cache/internal/peers"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
	"google.golang.org/protobuf/proto"
)

const (
	defaultBasePath = "/_gocache/"
	defaultReplicas = 50
)

// Protocol defines the communication protocol for peer communication
type Protocol string

const (
	// ProtocolHTTP indicates HTTP protocol
	ProtocolHTTP Protocol = "http"

	// ProtocolProtobuf indicates protobuf over HTTP
	ProtocolProtobuf Protocol = "protobuf"
)

// HTTPPool implements the server side of the distributed cache protocol
type HTTPPool struct {
	self          string                 // this peer's URL (host:port)
	basePath      string                 // base path of HTTP requests
	mu            sync.RWMutex           // guards peers and httpGetters
	peers         *consistenthash.Map    // consistent hash map for peer selection
	httpGetters   map[string]*HTTPGetter // keyed by peer URL
	protocol      Protocol               // communication protocol
	serverCancels []context.CancelFunc   // list of cancel functions for server shutdown
}

// NewHTTPPool initializes an HTTP pool of peers
func NewHTTPPool(self string, opts ...HTTPPoolOption) *HTTPPool {
	pool := &HTTPPool{
		self:        self,
		basePath:    defaultBasePath,
		protocol:    ProtocolProtobuf, // Use protobuf by default
		httpGetters: make(map[string]*HTTPGetter),
	}

	for _, opt := range opts {
		opt(pool)
	}

	return pool
}

// HTTPPoolOption configures an HTTPPool
type HTTPPoolOption func(*HTTPPool)

// WithBasePath configures the HTTPPool base path
func WithBasePath(basePath string) HTTPPoolOption {
	return func(p *HTTPPool) {
		p.basePath = basePath
	}
}

// WithProtocol configures the HTTPPool protocol
func WithProtocol(protocol Protocol) HTTPPoolOption {
	return func(p *HTTPPool) {
		p.protocol = protocol
	}
}

// ServeHTTP handles all HTTP requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log the request
	logger.Debugf("[Server %s] %s %s", p.self, r.Method, r.URL.Path)

	// Check if the request path starts with the expected base path
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		http.Error(w, "unexpected path: "+r.URL.Path, http.StatusBadRequest)
		return
	}

	switch p.protocol {
	case ProtocolHTTP:
		p.handleHTTP(w, r)
	case ProtocolProtobuf:
		p.handleProtobuf(w, r)
	default:
		http.Error(w, "unsupported protocol", http.StatusInternalServerError)
	}
}

// handleHTTP handles traditional HTTP GET requests
func (p *HTTPPool) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request path: /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request format", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// Get the cache group
	group := cache.GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// Get the value
	view, err := group.Get(key)
	if err != nil {
		if cache.IsKeyEmptyError(err) {
			http.Error(w, "key is empty", http.StatusBadRequest)
		} else if cache.IsKeyNotFoundError(err) {
			http.Error(w, fmt.Sprintf("key '%s' not found", key), http.StatusNotFound)
		} else {
			logger.Errorf("获取数据错误: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Set Content-Type and write response
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// handleProtobuf handles protobuf requests
func (p *HTTPPool) handleProtobuf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the protobuf request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading request: "+err.Error(), http.StatusBadRequest)
		return
	}

	req := &pb.Request{}
	if err := proto.Unmarshal(body, req); err != nil {
		http.Error(w, "error unmarshaling request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the cache group
	group := cache.GetGroup(req.Group)
	if group == nil {
		http.Error(w, "no such group: "+req.Group, http.StatusNotFound)
		return
	}

	// Get the value
	view, err := group.Get(req.Key)
	if err != nil {
		if cache.IsKeyEmptyError(err) {
			http.Error(w, "key is empty", http.StatusBadRequest)
		} else if cache.IsKeyNotFoundError(err) {
			http.Error(w, fmt.Sprintf("key '%s' not found", req.Key), http.StatusNotFound)
		} else {
			logger.Errorf("获取数据错误: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Create and marshal the response
	resp := &pb.Response{
		Value: view.ByteSlice(),
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, "error marshaling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set Content-Type and write response
	w.Header().Set("Content-Type", "application/protobuf")
	w.Write(data)
}

// Set updates the pool's peers
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create consistent hash map
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)

	// Create HTTP clients for each peer
	for _, peer := range peers {
		if peer != p.self { // Don't create a client to ourselves
			p.httpGetters[peer] = NewHTTPGetter(peer + p.basePath)
		}
	}

	logger.Infof("Cache pool set %d peers: %v", len(peers), peers)
}

// PickPeer picks a peer according to key
func (p *HTTPPool) PickPeer(key string) (peers.PeerGetter, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.peers == nil {
		return nil, false
	}

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		logger.Debugf("Pick peer %s for key %s", peer, key)
		return p.httpGetters[peer], true
	}

	return nil, false
}

// Start starts the HTTP server
func (p *HTTPPool) Start(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	server := &http.Server{
		Addr:    addr,
		Handler: p,
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.serverCancels = append(p.serverCancels, cancel)
	p.mu.Unlock()

	logger.Infof("Cache server started on %s", addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Cache server error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		logger.Info("Shutting down cache server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultClientTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("Error shutting down cache server: %v", err)
		}
	}()

	return nil
}

// Stop stops all HTTP servers
func (p *HTTPPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, cancel := range p.serverCancels {
		cancel()
	}

	p.serverCancels = nil
}

// Ensure HTTPPool implements peers.PeerPicker
var _ peers.PeerPicker = (*HTTPPool)(nil)

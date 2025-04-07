package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
	"google.golang.org/protobuf/proto"
)

const (
	defaultClientTimeout = 5 * time.Second
)

// HTTPGetter is a client to fetch cache data from peer
type HTTPGetter struct {
	baseURL string        // base URL of the remote server
	client  *http.Client  // HTTP client for making requests
	timeout time.Duration // timeout for HTTP requests
}

// NewHTTPGetter creates a new HTTP client for fetching cache data
func NewHTTPGetter(baseURL string) *HTTPGetter {
	return &HTTPGetter{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: defaultClientTimeout,
		},
		timeout: defaultClientTimeout,
	}
}

// Get fetches data from a peer using HTTP
func (h *HTTPGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v/%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get from peer: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, cache.ErrNotFound
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned non-200 status: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return bytes, nil
}

// GetByProto fetches data from peer using Protocol Buffers
func (h *HTTPGetter) GetByProto(req *pb.Request, resp *pb.Response) error {
	// Serialize the request to protobuf
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Create HTTP request
	u := h.baseURL
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/protobuf")

	// Execute request
	httpResp, err := h.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to get from peer: %w", err)
	}
	defer httpResp.Body.Close()

	// Check response status
	if httpResp.StatusCode == http.StatusNotFound {
		return cache.ErrNotFound
	} else if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer returned non-200 status: %v", httpResp.Status)
	}

	// Read and parse response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Unmarshal response
	if err = proto.Unmarshal(respBody, resp); err != nil {
		logger.Errorf("Failed to unmarshal response: %v", err)
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// SetTimeout sets the HTTP client timeout
func (h *HTTPGetter) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
	h.client.Timeout = timeout
}

package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

// httpGetter 用于从HTTP服务器获取数据
type httpGetter struct {
	baseURL string
}

// Get 从HTTP服务器获取数据
//
// 传入参数:
//   - group: 组名
//   - key: 键名
//
// 返回值:
//   - 数据: []byte
//   - 错误: error
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	logrus.Infof("[httpGetter] Get %s/%s from %s", group, key, h.baseURL)

	// 最终要访问的HTTP服务器的完整URL
	url := fmt.Sprintf("%v%v/%v", h.baseURL, group, key)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 如果HTTP状态码不是200，则返回错误
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", resp.Status)
	}

	// 读取响应体
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

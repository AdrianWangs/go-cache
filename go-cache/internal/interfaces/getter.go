package interfaces

// Getter 定义了获取缓存的方法
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 实现了Getter接口的函数类型
type GetterFunc func(key string) ([]byte, error)

// Get 实现Getter接口
//
// 传入参数:
//   - key: 缓存的key
//
// 返回值:
//   - []byte: 缓存的value
//   - error: 错误信息
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

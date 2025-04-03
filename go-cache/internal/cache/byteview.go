package cache

// ByteView 只读数据
// 实现了Value接口
type ByteView struct {
	bytes []byte
}

// Len 获取数据的长度
//
// 返回值:
//   - int: 数据的长度
func (v ByteView) Len() int {
	return len(v.bytes)
}

// ByteSlice 返回数据的副本
//
// 返回值:
//   - []byte: 数据的副本
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.bytes)
}

// cloneBytes 返回数据的副本
//
// 传入参数:
//   - bytes: 数据
//
// 返回值:
//   - []byte: 数据的副本
func cloneBytes(bytes []byte) []byte {
	c := make([]byte, len(bytes))
	copy(c, bytes)
	return c
}

// String 返回数据的字符串表示
//
// 返回值:
//   - string: 数据的字符串表示
func (v ByteView) String() string {
	return string(v.bytes)
}

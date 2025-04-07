package cache

import (
	"errors"
	"fmt"
)

// 错误类型枚举
const (
	// ErrTypeNone 无错误
	ErrTypeNone = iota
	// ErrTypeKeyEmpty 键为空
	ErrTypeKeyEmpty
	// ErrTypeKeyNotFound 键不存在
	ErrTypeKeyNotFound
	// ErrTypeGroupNotFound 组不存在
	ErrTypeGroupNotFound
	// ErrTypeInternalError 内部错误
	ErrTypeInternalError
	// ErrTypeNetworkError 网络错误
	ErrTypeNetworkError
)

// 预定义的错误
var (
	// ErrEmptyKey 表示键为空
	ErrEmptyKey = NewCacheError(ErrTypeKeyEmpty, "key is empty")
	// ErrNotFound 表示键不存在
	ErrNotFound = NewCacheError(ErrTypeKeyNotFound, "key not found")
	// ErrNoSuchGroup 表示缓存组不存在
	ErrNoSuchGroup = NewCacheError(ErrTypeGroupNotFound, "cache group not found")
)

// CacheError 表示缓存错误
type CacheError struct {
	Type    int    // 错误类型
	Message string // 错误信息
	Cause   error  // 原始错误（可选）
}

// Error 实现error接口
func (e *CacheError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 实现errors.Unwrap接口，支持错误链
func (e *CacheError) Unwrap() error {
	return e.Cause
}

// NewCacheError 创建一个新的缓存错误
func NewCacheError(errType int, message string) *CacheError {
	return &CacheError{
		Type:    errType,
		Message: message,
	}
}

// WrapError 包装一个错误
func WrapError(errType int, message string, cause error) *CacheError {
	return &CacheError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// IsKeyEmptyError 判断是否为键为空错误
func IsKeyEmptyError(err error) bool {
	var cacheErr *CacheError
	return errors.As(err, &cacheErr) && cacheErr.Type == ErrTypeKeyEmpty
}

// IsKeyNotFoundError 判断是否为键不存在错误
func IsKeyNotFoundError(err error) bool {
	var cacheErr *CacheError
	return errors.As(err, &cacheErr) && cacheErr.Type == ErrTypeKeyNotFound
}

// IsGroupNotFoundError 判断是否为组不存在错误
func IsGroupNotFoundError(err error) bool {
	var cacheErr *CacheError
	return errors.As(err, &cacheErr) && cacheErr.Type == ErrTypeGroupNotFound
}

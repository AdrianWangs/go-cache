package singleflight

import (
	"sync"

	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// call 代表正在进行中, 或已经结束的请求,使用sync.WaitGroup来确保请求的唯一性
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 管理不同key的请求(call)
type Group struct {
	mu    sync.Mutex
	calls map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {

	// 确保对call的map操作是安全的
	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[string]*call)
	}
	// 如果对于当前的key的请求正在执行, 则等待
	// 相当于第一个线程负责真正的对fn的执行
	// 其他线程等待第一个线程执行完毕，直接获取他们的结果
	if c, ok := g.calls[key]; ok {
		logger.Info("singleflight: Do() wait for ", key)
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	// 创建一个新的call
	c := new(call)
	c.wg.Add(1)
	g.calls[key] = c
	g.mu.Unlock()

	// 执行fn, 获取返回值
	c.val, c.err = fn()

	c.wg.Done()
	// 请求执行完毕, 删除call
	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()

	return c.val, c.err
}

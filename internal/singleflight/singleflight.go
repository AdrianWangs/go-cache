// Package singleflight provides a duplicate function call suppression mechanism
package singleflight

import (
	"context"
	"sync"
)

// call represents an in-flight or completed Do call
type call struct {
	wg    sync.WaitGroup // used to wait for the call to complete
	val   interface{}    // result of the call
	err   error          // error from the call
	ctx   context.Context
	ready chan struct{} // closed when val is ready
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Result holds the results of a Do call
type Result struct {
	Val    interface{}
	Err    error
	Shared bool // whether the result is being shared with other callers
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a time.
// If a duplicate call comes in, it will block until the original call completes
// and then return the same results.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	c.ready = make(chan struct{})
	g.m[key] = c
	g.mu.Unlock()

	// Execute the function
	go g.doCall(key, c, fn)

	// Wait for the call to complete
	c.wg.Wait()
	return c.val, c.err
}

// doCall executes the call and signals completion to any waiting callers
func (g *Group) doCall(key string, c *call, fn func() (interface{}, error)) {
	defer func() {
		// Remove the call from the map when done
		g.mu.Lock()
		delete(g.m, key)
		g.mu.Unlock()
		c.wg.Done()
	}()

	// Execute the function
	c.val, c.err = fn()
}

// DoChan is like Do but returns a channel that will receive the
// results when they are ready.
func (g *Group) DoChan(key string, fn func() (interface{}, error)) <-chan Result {
	ch := make(chan Result, 1)
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		go func() {
			c.wg.Wait()
			ch <- Result{c.val, c.err, true}
		}()
		return ch
	}
	c := new(call)
	c.wg.Add(1)
	c.ready = make(chan struct{})
	g.m[key] = c
	g.mu.Unlock()

	go func() {
		c.val, c.err = fn()
		c.wg.Done()
		ch <- Result{c.val, c.err, false}
		g.mu.Lock()
		delete(g.m, key)
		g.mu.Unlock()
	}()

	return ch
}

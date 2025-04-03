package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestConcurrentAPIAccess 测试并发访问 API
func TestConcurrentAPIAccess(t *testing.T) {
	// 设置并发请求数
	concurrentRequests := 100
	url := "http://localhost:9999/api?key=Tom"

	// 用于等待所有 goroutine 完成
	var wg sync.WaitGroup
	wg.Add(concurrentRequests)

	// 用于同步所有 goroutine 同时开始
	ready := make(chan struct{})

	// 用于收集响应结果
	results := make(chan string, concurrentRequests)
	errors := make(chan error, concurrentRequests)

	// 启动多个 goroutine 同时发起请求
	for i := 0; i < concurrentRequests; i++ {
		go func(id int) {
			defer wg.Done()

			// 等待信号以同时开始请求
			<-ready

			// 发起 HTTP GET 请求
			resp, err := http.Get(url)
			if err != nil {
				errors <- fmt.Errorf("request %d failed: %v", id, err)
				return
			}
			defer resp.Body.Close()

			// 读取响应
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errors <- fmt.Errorf("reading response %d failed: %v", id, err)
				return
			}

			// 将响应添加到结果通道
			results <- fmt.Sprintf("Request %d: %s (Status: %s)", id, string(body), resp.Status)
		}(i)
	}

	// 关闭 ready 通道以触发所有 goroutine 同时开始
	fmt.Println("Starting concurrent requests...")
	close(ready)

	// 设置超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待所有请求完成或超时
	select {
	case <-done:
		// 处理结果和错误
		close(results)
		close(errors)

		// 统计成功和失败的请求
		successCount := 0
		errorCount := 0

		// 输出成功请求
		for result := range results {
			fmt.Println(result)
			successCount++
		}

		// 输出错误
		for err := range errors {
			fmt.Println(err)
			errorCount++
		}

		fmt.Printf("Completed: %d successful, %d failed\n", successCount, errorCount)

	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out after 30 seconds")
	}
}

// TestMultipleKeysAccess 测试访问多个不同的键
func TestMultipleKeysAccess(t *testing.T) {
	// 设置并发请求数
	concurrentRequests := 30
	keys := []string{"Tom", "Jack", "Sam"}

	// 用于等待所有 goroutine 完成
	var wg sync.WaitGroup
	wg.Add(concurrentRequests)

	// 用于同步所有 goroutine 同时开始
	ready := make(chan struct{})

	// 用于收集响应结果
	results := make(chan string, concurrentRequests)
	errors := make(chan error, concurrentRequests)

	// 启动多个 goroutine 同时发起请求
	for i := 0; i < concurrentRequests; i++ {
		go func(id int) {
			defer wg.Done()

			// 等待信号以同时开始请求
			<-ready

			// 选择一个键
			key := keys[id%len(keys)]
			url := fmt.Sprintf("http://localhost:9999/api?key=%s", key)

			// 发起 HTTP GET 请求
			resp, err := http.Get(url)
			if err != nil {
				errors <- fmt.Errorf("request %d failed: %v", id, err)
				return
			}
			defer resp.Body.Close()

			// 读取响应
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errors <- fmt.Errorf("reading response %d failed: %v", id, err)
				return
			}

			// 将响应添加到结果通道
			results <- fmt.Sprintf("Request %d for key %s: %s (Status: %s)",
				id, key, string(body), resp.Status)
		}(i)
	}

	// 关闭 ready 通道以触发所有 goroutine 同时开始
	fmt.Println("Starting concurrent requests with multiple keys...")
	close(ready)

	// 设置超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待所有请求完成或超时
	select {
	case <-done:
		// 处理结果和错误
		close(results)
		close(errors)

		// 统计成功和失败的请求
		successCount := 0
		errorCount := 0

		// 输出成功请求
		for result := range results {
			fmt.Println(result)
			successCount++
		}

		// 输出错误
		for err := range errors {
			fmt.Println(err)
			errorCount++
		}

		fmt.Printf("Completed: %d successful, %d failed\n", successCount, errorCount)

	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out after 30 seconds")
	}
}

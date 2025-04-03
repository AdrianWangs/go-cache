package cache

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/AdrianWangs/go-cache/internal/interfaces"
	"github.com/AdrianWangs/go-cache/pkg/logger"
)

func TestGetter(t *testing.T) {
	// 初始化日志
	logger.InitLogger("info")

	var f interfaces.Getter = interfaces.GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Fatalf("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

var loadCounts = make(map[string]int, len(db))

func TestGet(t *testing.T) {
	// 初始化日志
	logger.InitLogger("debug")

	gee := NewGroup("scores", 2<<10, interfaces.GetterFunc(
		func(key string) ([]byte, error) {
			logger.Debugf("[SlowDB] search key %s", key)
			if v, ok := db[key]; ok {
				// 记录缓存未命中次数
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))

	for k, v := range db {
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of %s", k)
		}
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	// 测试缓存未命中，如果不报错，则逻辑错误
	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}

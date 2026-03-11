package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	// 模拟 50 个用户同时冲进来
	numRequests := 50
	url := "http://localhost:9999/api?key=Tom"

	fmt.Printf("🚀 开始并发测试，同时发送 %d 个请求...\n", numRequests)
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("❌ 请求 %d 失败: %v\n", id, err)
				return
			}
			defer resp.Body.Close()
			// 这里可以打印结果，确保大家都拿到了正确的数据
		}(i)
	}

	wg.Wait()
	fmt.Printf("✅ 所有请求已完成，总耗时: %v\n", time.Since(start))
}

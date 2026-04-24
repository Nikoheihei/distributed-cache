package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	baseURL := getenv("BASE_URL", "http://localhost:9999")
	url := fmt.Sprintf("%s/api?key=Tom", baseURL)
	numRequests := getenvInt("CONCURRENCY", 50)

	var wg sync.WaitGroup
	start := time.Now()

	fmt.Printf("start smoke test: %d concurrent requests\n", numRequests)
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("request %d failed: %v\n", id, err)
				return
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("request %d failed: status=%d\n", id, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("done: %v\n", time.Since(start))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

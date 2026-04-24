package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var wg sync.WaitGroup
	startTime := time.Now()

	baseURL := getenv("BASE_URL", "http://localhost:9999")
	concurrency := getenvInt("CONCURRENCY", 50)
	perWorker := getenvInt("PER_WORKER", 200)
	keySpace := getenvInt("KEY_SPACE", 10000)
	zipfS := getenvFloat("ZIPF_S", 0) // >0 enables Zipf
	zipfV := getenvFloat("ZIPF_V", 1)
	reqTimeout := getenvDuration("REQ_TIMEOUT", 3*time.Second)
	totalTimeout := getenvDuration("TOTAL_TIMEOUT", 2*time.Minute)

	if keySpace <= 0 {
		keySpace = 1
	}

	client := &http.Client{
		Timeout: reqTimeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	var okCount atomic.Uint64
	var failCount atomic.Uint64
	var firstErrMu sync.Mutex
	var firstErr error

	// 开启并发协程（模拟真实用户同时访问）
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			seed := time.Now().UnixNano() + int64(workerID)*1000003
			rng := rand.New(rand.NewSource(seed))
			var zipf *rand.Zipf
			if zipfS > 0 && keySpace > 1 {
				zipf = rand.NewZipf(rng, zipfS, zipfV, uint64(keySpace-1))
			}

			// 每个用户随机查询不同的 Key
			for j := 0; j < perWorker; j++ {
				if ctx.Err() != nil {
					return
				}

				var id int
				if zipf != nil {
					id = int(zipf.Uint64()) + 1 // [1, keySpace]
				} else {
					// 默认均匀分布，避免“顺序扫库”带来的不真实 miss 模式。
					id = rng.Intn(keySpace) + 1
				}

				key := fmt.Sprintf("User%d", id)
				url := fmt.Sprintf("%s/api?key=%s", baseURL, key)

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					recordFail(&failCount, &firstErrMu, &firstErr, err)
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					recordFail(&failCount, &firstErrMu, &firstErr, err)
					continue
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					recordFail(&failCount, &firstErrMu, &firstErr, fmt.Errorf("status=%d", resp.StatusCode))
					continue
				}
				okCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	if ctx.Err() != nil {
		fmt.Printf("压测超时退出: %v\n", ctx.Err())
	}
	if firstErr != nil {
		fmt.Printf("首个错误: %v\n", firstErr)
	}
	fmt.Printf("压测完成 | ok=%d fail=%d | 总耗时: %v\n", okCount.Load(), failCount.Load(), time.Since(startTime))
}

func recordFail(failCount *atomic.Uint64, firstErrMu *sync.Mutex, firstErr *error, err error) {
	failCount.Add(1)
	firstErrMu.Lock()
	if *firstErr == nil {
		*firstErr = err
	}
	firstErrMu.Unlock()
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

func getenvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

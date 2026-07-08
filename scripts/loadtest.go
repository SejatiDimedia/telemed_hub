package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

func main() {
	targetURL := flag.String("url", "http://localhost:8080/readyz", "URL to hit for load testing")
	concurrency := flag.Int("c", 20, "Number of concurrent workers")
	totalRequests := flag.Int("n", 2000, "Total number of requests to execute")
	flag.Parse()

	fmt.Printf("Starting load test against: %s\n", *targetURL)
	fmt.Printf("Concurrency: %d workers, Total Requests: %d\n", *concurrency, *totalRequests)

	transport := &http.Transport{
		MaxIdleConnsPerHost: *concurrency,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	chRequests := make(chan struct{}, *totalRequests)
	for i := 0; i < *totalRequests; i++ {
		chRequests <- struct{}{}
	}
	close(chRequests)

	var wg sync.WaitGroup
	latencies := make([]time.Duration, *totalRequests)
	var mu sync.Mutex
	idx := 0

	successCount := 0
	errorCount := 0

	startTime := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range chRequests {
				reqStart := time.Now()
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, *targetURL, nil)
				resp, err := client.Do(req)
				duration := time.Since(reqStart)

				mu.Lock()
				if err != nil || resp.StatusCode != http.StatusOK {
					errorCount++
					if resp != nil {
						resp.Body.Close()
					}
				} else {
					successCount++
					resp.Body.Close()
					latencies[idx] = duration
					idx++
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	// Filter out zero latencies from errors
	actualLatencies := latencies[:successCount]
	sort.Slice(actualLatencies, func(i, j int) bool {
		return actualLatencies[i] < actualLatencies[j]
	})

	rps := float64(successCount+errorCount) / totalTime.Seconds()

	fmt.Println("\nBenchmark Report:")
	fmt.Printf("--------------------------------------------\n")
	fmt.Printf("Total Time Elapsed:  %v\n", totalTime)
	fmt.Printf("Successful Requests: %d\n", successCount)
	fmt.Printf("Failed Requests:     %d\n", errorCount)
	fmt.Printf("Requests Per Second: %.2f RPS\n", rps)

	if successCount > 0 {
		p50 := actualLatencies[int(float64(successCount)*0.5)]
		p90 := actualLatencies[int(float64(successCount)*0.90)]
		p95 := actualLatencies[int(float64(successCount)*0.95)]

		fmt.Printf("P50 (Median) Latency: %v\n", p50)
		fmt.Printf("P90 Latency:          %v\n", p90)
		fmt.Printf("P95 Latency:          %v\n", p95)
		fmt.Printf("--------------------------------------------\n")

		if p95 < 300*time.Millisecond {
			fmt.Printf("\033[32mSUCCESS: P95 Latency (%.2fms) is below 300ms target!\033[0m\n", float64(p95.Microseconds())/1000.0)
		} else {
			fmt.Printf("\033[31mWARNING: P95 Latency (%.2fms) exceeded 300ms target!\033[0m\n", float64(p95.Microseconds())/1000.0)
		}
	} else {
		fmt.Println("No successful requests recorded.")
	}
}

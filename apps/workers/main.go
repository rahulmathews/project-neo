package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck("http://localhost:8083/health"))
	}

	if err := run(); err != nil {
		log.Fatalf("workers: %v", err)
	}
}

func runHealthcheck(url string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("healthcheck failed: %v", err)
		return 1
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("healthcheck close body error: %v", closeErr)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("healthcheck failed: unexpected status %d", resp.StatusCode)
		return 1
	}

	fmt.Println("ok")
	return 0
}

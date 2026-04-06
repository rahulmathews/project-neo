package main

import (
	"log"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("workers: %v", err)
	}
}

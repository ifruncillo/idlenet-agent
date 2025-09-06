package main

import (
	"fmt"
	"time"
)

const version = "0.1.0"

func main() {
	fmt.Println("IdleNet Agent (stub) v" + version)
	fmt.Println("This build is for onboarding only. It does NOT run any jobs.")
	fmt.Println("Press Ctrl+C to quit.")
	for {
		fmt.Println(time.Now().Format(time.RFC3339), "heartbeat")
		time.Sleep(30 * time.Second)
	}
}

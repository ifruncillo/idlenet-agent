package main

import (
    "fmt"
    "time"
)

func main() {
    fmt.Println("IdleNet Agent v1.0.0")
    fmt.Println("This is a simple test version")
    
    // Simple loop that prints a message every 5 seconds
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    fmt.Println("Running... Press Ctrl+C to stop")
    
    for range ticker.C {
        fmt.Println(time.Now().Format("15:04:05"), "- Still running")
    }
}
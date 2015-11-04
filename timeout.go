// Go Concurrency Pattern: Timeout
//
package main

import (
    "fmt"
    "math/rand"
    "time"
)

func demoWorker() <-chan int {
    cost := make(chan int)
    go func() {
        // sleep rand 2 ~ 6 seconds as if it is working
        t := 2 + rand.Int() % 5
        time.Sleep(time.Duration(t) * time.Second)
        cost <- t
    }()
    return cost
}

func timeoutWorker(n int) <-chan struct{} {
    timeout := make(chan struct{})
    go func() {
        // sleep n seconds for timeout
        time.Sleep(time.Duration(n) * time.Second)
        timeout <- struct{}{}
    }()
    return timeout
}

func main() {
    rand.Seed(time.Now().Unix())

    n := 5
    select {
    case cost := <-demoWorker():
        fmt.Printf("Work Complete, cost: %d seconds\n", cost)
    case <-timeoutWorker(n):
        fmt.Printf("Timeout after %d seconds\n", n)
    }
}

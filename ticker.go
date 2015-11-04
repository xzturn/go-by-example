// Go Concurrency Pattern: Ticker
//
package main

import (
    "fmt"
    "time"
)

func workProcess(idx *int) {
    fmt.Printf("Working [%d] ... ...\n", *idx)
    *idx++
}

func PeriodicWorker(interval time.Duration) (<-chan int, chan<- struct{}) {
    times, quit := make(chan int), make(chan struct{})
    ticker, idx := time.NewTicker(interval), 0
    go func() {
        defer close(times)   // stop the main routine's wating
        for {
            select {
            case <-ticker.C: // trigger by ticker
                workProcess(&idx)
                times <- idx
            case <-quit:     // stop the closure go routine
                return
            }
        }
    }()
    return times, quit
}

func main() {
    t, n := time.Duration(1) * time.Second, 10
    times, quit := PeriodicWorker(t)
    for i := range times {   // main routine wating on times
        if i >= n {          // loop n times, then send quit signal
            quit <- struct{}{}
        }
    }
}

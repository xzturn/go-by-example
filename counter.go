// Atomic Counters accessed by mulitple go routines
//
package main

import (
    "fmt"
    "runtime"
    "sync/atomic"
    "time"
)

func main() {
    var ops uint64 = 0

    done, n := make(chan struct{}), 100

    for i := 0; i < n; i++ {
        go func() {
            for {
                select {
                case <-done:
                    return
                default:
                    atomic.AddUint64(&ops, 1)
                    runtime.Gosched()
                }
            }
        }()
    }

    time.Sleep(time.Second)

    for i := 0; i < n; i++ { done <- struct{}{} }

    opsFinal := atomic.LoadUint64(&ops)
    fmt.Println("counter:", opsFinal)
}

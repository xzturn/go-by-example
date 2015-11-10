// Go crontab implementation using ticker
//
package main

import (
    "fmt"
    "time"
)

func workProcess() {
    fmt.Printf("[%v] Working ... ...\n", time.Now())
}

func work(interval time.Duration, worker chan<- struct{}) {
    ticker := time.NewTicker(interval)
    for {
        select {
        case <-ticker.C:    // trigger by interval ticker
            workProcess()
            worker <- struct{}{}
        }
    }
}

func main() {
    n := 10                 // interval in minutes
    delta, tnow := time.Duration(n) * time.Minute, time.Now()
    t := time.Date(tnow.Year(), tnow.Month(), tnow.Day(), tnow.Hour(), tnow.Minute() / n * n, 0, 0, time.Local).Add(delta)
    d, worker := t.Sub(tnow), make(chan struct{})
    ts := <-time.After(d)   // trigger first launch at xx:[0-5]0
    fmt.Printf("[%v] First working process start at %v.\n", time.Now(), ts)
    go work(delta, worker)  // start the ticker worker
    for {
        select {
        case <-worker:      // the main go routine wait on worker, won't exit
            fmt.Printf("[%v] Work complete!\n", time.Now())
        }
    }
}

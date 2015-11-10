// Go singal process
//
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
)

func sigWorker() (chan os.Signal, <-chan struct{}) {
    csig := make(chan os.Signal)
    done := make(chan struct{})
    go func() {
        // wait the SIGINT/SIGTERM signal
        fmt.Printf("\n%v\n", <-csig)
        done <- struct{}{}
    }()
    return csig, done
}

func main() {
    csig, done := sigWorker()

    // signal.Notify registers the given channel to
    // receive notifications of the spec signals
    signal.Notify(csig, syscall.SIGINT, syscall.SIGTERM)
    fmt.Println("awaiting signal")

    <-done // main go routine waits here
    fmt.Println("exiting")
}

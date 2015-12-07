// Run like crontab:
//    Start at hhmmss and run by interval
//
package main

import (
    "encoding/json"
    "errors"
    "flag"
    "io"
    "io/ioutil"
    "log"
    "os"
    "runtime"
    "strconv"
    "strings"
    "time"
)

////////////////////////////////////////////////////////////////////////////////

var configFile  *string = flag.String("c", ``, "config file which specify all params")
var startTime   *string = flag.String("s", ``, "specify start time: ${hh:mm:ss}")
var intervalSec *int = flag.Int("i", -1, "intervals in seconds, should > 0")


////////////////////////////////////////////////////////////////////////////////

func parseHourMinSec(hms string) (h, m, s int, e error) {
    tmp := strings.Split(hms, ":")
    if len(tmp) != 3 {
        e = errors.New("expect hh:mm:ss")
        return
    }

    h, e = strconv.Atoi(strings.TrimPrefix(tmp[0], "0"))
    if e != nil { return }
    if h < 0 || h > 23 {
        e = errors.New("expect 00 <= hh <= 23")
        return
    }

    m, e = strconv.Atoi(strings.TrimPrefix(tmp[1], "0"))
    if e != nil { return }
    if m < 0 || s > 59 {
        e = errors.New("expect 00 <= mm <= 59")
        return
    }

    s, e = strconv.Atoi(strings.TrimPrefix(tmp[2], "0"))
    if e != nil { return }
    if s < 0 || s > 59 {
        e = errors.New("expect 00 <= ss <= 59")
        return
    }

    return
}

////////////////////////////////////////////////////////////////////////////////

type PeriodicConfig struct {
    StartTime string  `json:"start_time"`
    Interval  int     `json:"interval_in_seconds"`
    LogFile   string  `json:"log_file"`
}

func (c *PeriodicConfig) ParseFromJsonFile(cfgFile string) error {
    blob, err := ioutil.ReadFile(cfgFile)
    if err != nil { return err }
    return json.Unmarshal(blob, c)
}


////////////////////////////////////////////////////////////////////////////////

type PeriodicRunner struct {
    hour     int
    minute   int
    second   int
    interval time.Duration
    counter  int
    *log.Logger
}

func NewPeriodicRunner() *PeriodicRunner {
    var logFile string
    var cfg PeriodicConfig
    err := cfg.ParseFromJsonFile(*configFile)
    if err == nil { logFile = cfg.LogFile }

    var w io.Writer = os.Stdout
    if fp, err := os.Create(logFile); err == nil { w = fp }
    logger := log.New(w, "PeriodicRunner: ", log.LstdFlags)

    stime := *startTime
    h, m, s, e := parseHourMinSec(stime)
    if e != nil {
        h, m, s, err = parseHourMinSec(cfg.StartTime)
        if err != nil {
            h, m, s = 0, 0, 0
        }
    }

    is := *intervalSec
    if is <= 0 { is = cfg.Interval }
    if is <= 0 { is = 86400 }
    i := time.Duration(is) * time.Second

    return &PeriodicRunner{h, m, s, i, 0, logger}
}

func (p *PeriodicRunner) run(done chan<- struct{}) {
    p.counter++
    p.Printf("[%d] PeriodicRunner running ... ...", p.counter)
    done <- struct{}{}
}

func (p *PeriodicRunner) worker(done chan<- struct{}) {
    go p.run(done)
    ticker := time.NewTicker(p.interval)
    for {
        select {
        case <-ticker.C:
            p.run(done)
        }
    }
}

func (p *PeriodicRunner) Run() {
    t, done := time.Now(), make(chan struct{})
    target := time.Date(t.Year(), t.Month(), t.Day(), p.hour, p.minute, p.second, 0, time.Local)
    if target.Sub(t) < 0 { target = target.Add(time.Duration(24) * time.Hour) }
    p.Printf("PeriodicRunner will start at %v", target)

    ts := <-time.After(target.Sub(t))
    p.Printf("First start at time %v", ts)
    go p.worker(done)

    for {
        select {
        case <-done:
            p.Printf("[%d] work complete!", p.counter)
        }
    }
}

////////////////////////////////////////////////////////////////////////////////
// package init & main
////////////////////////////////////////////////////////////////////////////////

func init() {
    runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
    flag.Parse()
    p := NewPeriodicRunner()
    p.Run()
}

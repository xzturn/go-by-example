// Go Concurrency Pattern: Pipelines and Cancellation & Worker Pool
//
package main

import (
    "crypto/md5"
    "errors"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "sort"
    "sync"
    "time"
)

var workType *int = flag.Int("t", 0, "FileDigester Type: 0, 1, 2, or 3")

// A result is the product of reading and summing a file using MD5.
type result struct {
    path string
    sum  [md5.Size]byte
    err  error
}

////////////////////////////////////////////////////////////////////////////////

type IFileDigester interface {
    MD5All(string) (map[string][md5.Size]byte, error)
}

////////////////////////////////////////////////////////////////////////////////

type FileDigester struct {
}

// walk through all the files and sub-dirs, no concurrency
func (p FileDigester) walk(root string, files *[]string) error {
    infos, err := ioutil.ReadDir(root)
    if err != nil { return err }
    for _, info := range infos {
        switch {
        case info.Mode().IsRegular():
            *files = append(*files, filepath.Join(root, info.Name()))
        case info.Mode().IsDir():
            if err = p.walk(filepath.Join(root, info.Name()), files); err != nil {
                return err
            }
        }
    }
    return nil
}

// define how each worker work, wait for idx signal or done signal
func (p FileDigester) md5Worker(files []string, cidx <-chan int, done <-chan struct{}, res *[]result) {
    for {
        select {
        case idx:= <-cidx:
            data, err := ioutil.ReadFile(files[idx])
            (*res)[idx] = result{files[idx], md5.Sum(data), err}
        case <-done:
            return
        }
    }
}

// worker pool: collect candicate files first,
// then use N go routines to process each file, use cancel
func (p FileDigester) MD5All(root string) (map[string][md5.Size]byte, error) {
    ts := time.Now()
    defer func() {
        fmt.Printf("FileDigester.MD5All() execute time: %v\n", time.Now().Sub(ts))
    }()

    files := make([]string, 0)
    if err := p.walk(root, &files); err != nil { return nil, err }

    n := len(files)
    cidx := make(chan int)
    done := make(chan struct{})
    res := make([]result, n)

    for i := 0; i < n; i++ { go p.md5Worker(files, cidx, done, &res) }
    for i := 0; i < n; i++ { cidx <- i }
    for i := 0; i < n; i++ { done <- struct{}{} }

    m := make(map[string][md5.Size]byte)
    for _, r := range res {
        if r.err != nil {
            return nil, r.err
        }
        m[r.path] = r.sum
    }
    return m, nil
}

////////////////////////////////////////////////////////////////////////////////

type FileDigester1 struct {
}

// sumFiles starts goroutines to walk the directory tree at root and digest each
// regular file.  These goroutines send the results of the digests on the result
// channel and send the result of the walk on the error channel.  If done is
// closed, sumFiles abandons its work.
func (p FileDigester1) sumFiles(done <-chan struct{}, root string) (<-chan result, <-chan error) {
    // For each regular file, start a goroutine that sums the file and sends
    // the result on c.  Send the result of the walk on errc.
    c := make(chan result)
    errc := make(chan error, 1)
    go func() { // HL
        var wg sync.WaitGroup
        err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            if !info.Mode().IsRegular() {
                return nil
            }
            wg.Add(1)
            go func() { // HL
                data, err := ioutil.ReadFile(path)
                select {
                case c <- result{path, md5.Sum(data), err}: // HL
                case <-done: // HL
                }
                wg.Done()
            }()
            // Abort the walk if done is closed.
            select {
            case <-done: // HL
                return errors.New("walk canceled")
            default:
                return nil
            }
        })
        // Walk has returned, so all calls to wg.Add are done.  Start a
        // goroutine to close c once all the sends are done.
        go func() { // HL
            wg.Wait()
            close(c) // HL
        }()
        // No select needed here, since errc is buffered.
        errc <- err // HL
    }()
    return c, errc
}

// MD5All reads all the files in the file tree rooted at root and returns a map
// from file path to the MD5 sum of the file's contents.  If the directory walk
// fails or any read operation fails, MD5All returns an error.  In that case,
// MD5All does not wait for inflight read operations to complete.
func (p FileDigester1) MD5All(root string) (map[string][md5.Size]byte, error) {
    ts := time.Now()
    defer func() {
        fmt.Printf("FileDigester1.MD5All() execute time: %v\n", time.Now().Sub(ts))
    }()
    // MD5All closes the done channel when it returns; it may do so before
    // receiving all the values from c and errc.
    done := make(chan struct{}) // HLdone
    defer close(done)           // HLdone

    c, errc := p.sumFiles(done, root) // HLdone

    m := make(map[string][md5.Size]byte)
    for r := range c { // HLrange
        if r.err != nil {
            return nil, r.err
        }
        m[r.path] = r.sum
    }
    if err := <-errc; err != nil {
        return nil, err
    }
    return m, nil
}

////////////////////////////////////////////////////////////////////////////////

type FileDigester2 struct {
}

// walkFiles starts a goroutine to walk the directory tree at root and send the
// path of each regular file on the string channel.  It sends the result of the
// walk on the error channel.  If done is closed, walkFiles abandons its work.
func (p FileDigester2) walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
    paths := make(chan string)
    errc := make(chan error, 1)
    go func() { // HL
        // Close the paths channel after Walk returns.
        defer close(paths) // HL
        // No select needed for this send, since errc is buffered.
        errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error { // HL
            if err != nil {
                return err
            }
            if !info.Mode().IsRegular() {
                return nil
            }
            select {
            case paths <- path: // HL
            case <-done: // HL
                return errors.New("walk canceled")
            }
            return nil
        })
    }()
    return paths, errc
}

// digester reads path names from paths and sends digests of the corresponding
// files on c until either paths or done is closed.
func (p FileDigester2) digester(done <-chan struct{}, paths <-chan string, c chan<- result) {
    for path := range paths { // HLpaths
        data, err := ioutil.ReadFile(path)
        select {
        case c <- result{path, md5.Sum(data), err}:
        case <-done:
            return
        }
    }
}

// MD5All reads all the files in the file tree rooted at root and returns a map
// from file path to the MD5 sum of the file's contents.  If the directory walk
// fails or any read operation fails, MD5All returns an error.  In that case,
// MD5All does not wait for inflight read operations to complete.
func (p FileDigester2) MD5All(root string) (map[string][md5.Size]byte, error) {
    ts := time.Now()
    defer func() {
        fmt.Printf("FileDigester2.MD5All() execute time: %v\n", time.Now().Sub(ts))
    }()
    // MD5All closes the done channel when it returns; it may do so before
    // receiving all the values from c and errc.
    done := make(chan struct{})
    defer close(done)

    paths, errc := p.walkFiles(done, root)

    // Start a fixed number of goroutines to read and digest files.
    c := make(chan result) // HLc
    var wg sync.WaitGroup
    const numDigesters = 20
    wg.Add(numDigesters)
    for i := 0; i < numDigesters; i++ {
        go func() {
            p.digester(done, paths, c) // HLc
            wg.Done()
        }()
    }
    go func() {
        wg.Wait()
        close(c) // HLc
    }()
    // End of pipeline. OMIT

    m := make(map[string][md5.Size]byte)
    for r := range c {
        if r.err != nil {
            return nil, r.err
        }
        m[r.path] = r.sum
    }
    // Check whether the Walk failed.
    if err := <-errc; err != nil { // HLerrc
        return nil, err
    }
    return m, nil
}

////////////////////////////////////////////////////////////////////////////////

type FileDigester3 struct {
}

// for each go routine, it will walk through a directory
func (p FileDigester3) walk(root string, cpath chan<- string) <-chan error {
    cerr := make(chan error)
    go func() {
        infos, err := ioutil.ReadDir(root)
        if err != nil { cerr <- err; return }
        for _, info := range infos {
            switch {
            case info.Mode().IsRegular():
                cpath <- filepath.Join(root, info.Name())
            case info.Mode().IsDir():
                err = <-p.walk(filepath.Join(root, info.Name()), cpath)
                if err != nil { cerr <- err; return }
            }
        }
        cerr <- nil
    }()
    return cerr
}

// process file by file actually, just collect in concurrent pattern
func (p FileDigester3) MD5All(root string) (map[string][md5.Size]byte, error) {
    ts := time.Now()
    defer func() {
        fmt.Printf("FileDigester3.MD5All() execute time: %v\n", time.Now().Sub(ts))
    }()

    cpath := make(chan string)
    cerr := p.walk(root, cpath)
    m := make(map[string][md5.Size]byte)
    for {
        select {
        case path := <-cpath:
            data, err := ioutil.ReadFile(path)
            if err != nil { return nil, err }
            m[path] = md5.Sum(data)
        case err := <-cerr:
            return m, err
        }
    }
}

////////////////////////////////////////////////////////////////////////////////

type FileDigester4 struct {
}

// walk through all the files and sub-dirs, collect all candidates
func (p FileDigester4) walk(root string, files *[]string) error {
    infos, err := ioutil.ReadDir(root)
    if err != nil { return err }
    for _, info := range infos {
        switch {
        case info.Mode().IsRegular():
            *files = append(*files, filepath.Join(root, info.Name()))
        case info.Mode().IsDir():
            if err = p.walk(filepath.Join(root, info.Name()), files); err != nil {
                return err
            }
        }
    }
    return nil
}

// define how each worker work, wait for cfile signal (buffered)
func (p FileDigester4) md5Worker(cfile <-chan string, cres chan<- result) {
    for file := range cfile {
        data, err := ioutil.ReadFile(file)
        cres <- result{file, md5.Sum(data), err}
    }
}

// worker pool: collect candicate files first,
// then use N go routines to process each file
func (p FileDigester4) MD5All(root string) (map[string][md5.Size]byte, error) {
    ts := time.Now()
    defer func() {
        fmt.Printf("FileDigester4.MD5All() execute time: %v\n", time.Now().Sub(ts))
    }()

    files := make([]string, 0)
    if err := p.walk(root, &files); err != nil { return nil, err }

    n := len(files)
    cfile := make(chan string, n)
    cres := make(chan result, n)

    for i := 0; i < n; i++ { go p.md5Worker(cfile, cres) }
    for i := 0; i < n; i++ { cfile <- files[i] }
    close(cfile)

    m := make(map[string][md5.Size]byte)
    for i := 0; i < n; i++ {
        r := <-cres
        if r.err != nil {
            return nil, r.err
        }
        m[r.path] = r.sum
    }
    close(cres)

    return m, nil
}

////////////////////////////////////////////////////////////////////////////////

func main() {
    flag.Parse()

    // Calculate the MD5 sum of all files under the specified directory,
    // then print the results sorted by path name.
    var p IFileDigester
    switch *workType {
    case 1:
        p = &FileDigester1{}
    case 2:
        p = &FileDigester2{}
    case 3:
        p = &FileDigester3{}
    case 4:
        p = &FileDigester4{}
    default:
        p = &FileDigester{}
    }

    root := "."
    if flag.NArg() > 0 {
        root = flag.Arg(0)
    }

    m, err := p.MD5All(root)
    if err != nil {
        fmt.Println(err)
        return
    }
    var paths []string
    for path := range m {
        paths = append(paths, path)
    }
    sort.Strings(paths)
    for _, path := range paths {
        fmt.Printf("%x  %s\n", m[path], path)
    }
}

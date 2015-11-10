// Json format config parse as json object
// Need define the json struct, or should reflect
//
package main

import (
    "encoding/json"
    "flag"
    "fmt"
)

var kConfigContent string = `{
    "server": "%s",
    "port": %d,
    "user": "%s",
    "password": "%s"
}`

var server   *string = flag.String("s", "smtp.example.com", "the smtp email server")
var port     *int    = flag.Int("p", 25, "the smtp email port")
var user     *string = flag.String("u", "my_username", "the email user name")
var password *string = flag.String("w", "my_password", "the email password")

type SmtpInfo struct {
    Server   string   `json:"server"`
    Port     int      `json:"port"`
    User     string   `json:"user"`
    Password string   `json:"password"`
}

func main() {
    flag.Parse()
    content := fmt.Sprintf(kConfigContent, *server, *port, *user, *password)
    fmt.Println(content)
    cfg := SmtpInfo{}
    if err := json.Unmarshal([]byte(content), &cfg); err != nil {
        fmt.Printf("Umarshal failed: %v\n", err)
    } else {
        fmt.Printf("%+v\n", cfg)
    }
}

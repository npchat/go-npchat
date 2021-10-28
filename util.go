package main

import (
	"flag"
	"net/http"

	"github.com/gorilla/websocket"
)

type Options struct {
	Port     int
	CertFile string
	KeyFile  string
}

type ServerMessage struct {
	Message string `json:"message"`
}

type ServerChallenge struct {
	Challenge Challenge `json:"challenge"`
}

type ClientMessage struct {
	Get       string    `json:"get"`
	Challenge Challenge `json:"challenge"`
	PublicKey string    `json:"publicKey"`
	Solution  string    `json:"solution"`
}

type ChatMessage struct {
	Id   string
	Body []byte
}

type Session struct {
	Id   string
	Conn *websocket.Conn
}

func GetOptionsFromFlags() Options {
	o := Options{}
	flag.StringVar(&o.CertFile, "cert", "", "must be a relative file path")
	flag.StringVar(&o.KeyFile, "key", "", "must be a relative file path")
	flag.IntVar(&o.Port, "p", PORT_HTTP, "port must be an int")
	flag.Parse()
	if o.CertFile != "" && o.KeyFile != "" && o.Port == PORT_HTTP {
		o.Port = PORT_HTTPS
	}
	return o
}

func CheckOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     CheckOrigin,
}

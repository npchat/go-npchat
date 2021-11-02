package main

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type User struct {
	Msgs   []Msg
	Conns  []Connection
	Online bool
	Mux    *sync.RWMutex
}

type Msg struct {
	Body []byte    // content
	Kick time.Time // time to kick from storage
}

type Connection struct {
	Sock *websocket.Conn
	Mux  *sync.Mutex
}

func (u *User) RegisterWebSocket(conn *websocket.Conn) {
	c := Connection{
		Sock: conn,
		Mux:  new(sync.Mutex),
	}
	u.Mux.Lock()
	u.Conns = append(u.Conns, c)
	u.Online = true
	u.Mux.Unlock()
}

func (u *User) UnregisterWebSocket(conn *websocket.Conn) {
	keep := []Connection{}
	u.Mux.Lock()
	for _, c := range u.Conns {
		if c.Sock != conn {
			keep = append(keep, c)
		}
	}
	if len(keep) < 1 {
		u.Online = false
	}
	u.Conns = keep
	u.Mux.Unlock()
}

func (u *User) Send(msg []byte, ttl time.Duration) {
	if u.Online {
		for _, c := range u.Conns {
			c.Mux.Lock()
			err := c.Sock.WriteMessage(websocket.TextMessage, msg)
			c.Mux.Unlock()
			if err != nil {
				log.Println(c.Sock.RemoteAddr(), "failed to send msg", err)
			}
		}
	} else { // offline, store it
		m := Msg{
			Body: msg,
			Kick: time.Now().Add(ttl),
		}
		u.Mux.Lock()
		u.Msgs = append(u.Msgs, m)
		u.Mux.Unlock()
	}
}

func (u *User) SendStored() {
	u.Mux.RLock()
	for _, c := range u.Conns {
		c.Mux.Lock()
		for _, m := range u.Msgs {
			err := c.Sock.WriteMessage(websocket.TextMessage, m.Body)
			if err != nil {
				log.Println(c.Sock.RemoteAddr(), "failed to send stored msg", err)
			}
		}
		c.Mux.Unlock()
	}
	u.Mux.RUnlock()
	// clear stored
	u.Mux.Lock()
	u.Msgs = []Msg{}
	u.Mux.Unlock()
}

func GetIdFromPath(path string) string {
	return strings.TrimLeft(path, "/")
}

package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type User struct {
	Id             string
	Msgs           []Msg
	Conns          []Connection  `json:"-"`
	Online         bool          `json:"-"`
	Mux            *sync.RWMutex `json:"-"`
	Pusher         Pusher
	Data           []byte
	LastConnection time.Time
}

type Msg struct {
	Body []byte    // content
	Kick time.Time // time to kick from storage
}

type MessageBody struct {
	From []byte `msgpack:"f"`
}

type Connection struct {
	Sock *websocket.Conn
	Mux  *sync.Mutex
}

/*
Type: "message"
From: "{publicKeyHash}"
type Notification struct {
	Type string
	From string
}*/

func (u *User) RegisterWebSocket(conn *websocket.Conn) {
	c := Connection{
		Sock: conn,
		Mux:  new(sync.Mutex),
	}
	u.Mux.Lock()
	u.Conns = append(u.Conns, c)
	u.Online = true
	u.LastConnection = time.Now()
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
	u.LastConnection = time.Now()
	u.Mux.Unlock()
}

func (u *User) Send(msg []byte, ttl time.Duration) {
	// store it
	m := Msg{
		Body: msg,
		Kick: time.Now().Add(ttl),
	}
	u.Mux.Lock()
	u.Msgs = append(u.Msgs, m)
	u.Mux.Unlock()
	if u.Online {
		for _, c := range u.Conns {
			c.Mux.Lock()
			err := c.Sock.WriteMessage(websocket.TextMessage, msg)
			c.Mux.Unlock()
			if err != nil {
				log.Println(c.Sock.RemoteAddr(), "failed to send msg", err)
			}
		}
	} else { // offline
		// send notification
		u.Pusher.Push(u.Id, []byte("You've got a message"))
		/*
			TODO: Unmarshal received msg, get 'from' id
			& serialise rich notification
		*/
	}
}

func (u *User) SendStored() {
	u.Mux.RLock()
	for _, c := range u.Conns {
		c.Mux.Lock()
		for _, m := range u.Msgs {
			err := c.Sock.WriteMessage(websocket.BinaryMessage, m.Body)
			if err != nil {
				log.Println(c.Sock.RemoteAddr(), "failed to send stored msg", err)
			}
		}
		c.Mux.Unlock()
	}
	u.Mux.RUnlock()
}

func (u *User) SetData(data []byte) {
	u.Mux.Lock()
	u.Data = data
	u.Mux.Unlock()
}

func (u *User) GetData() []byte {
	u.Mux.RLock()
	defer u.Mux.RUnlock()
	return u.Data
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shamaton/msgpack/v2"
)

type User struct {
	Msgs           []Msg
	Conns          []Connection  `json:"-"`
	Online         bool          `json:"-"`
	Mux            *sync.RWMutex `json:"-"`
	Pusher         Pusher
	Data           []byte
	ShareableData  []byte
	LastConnection time.Time
}

type Msg struct {
	Body []byte    // content
	Kick time.Time // time to kick from storage
}

type Connection struct {
	Sock *websocket.Conn
	Mux  *sync.Mutex
}

type MsgData struct {
	From []byte `msgpack:"f"`
}

type MsgPushNotification struct {
	Type string `json:"type"`
	From string `json:"from"`
}

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

func (u *User) Send(msg []byte, ttl time.Duration, doStore bool) {
	if doStore {
		// store it
		m := Msg{
			Body: msg,
			Kick: time.Now().Add(ttl),
		}
		u.Mux.Lock()
		u.Msgs = append(u.Msgs, m)
		u.Mux.Unlock()
	}

	if u.Online {
		for _, c := range u.Conns {
			c.Mux.Lock()
			err := c.Sock.WriteMessage(websocket.BinaryMessage, msg)
			c.Mux.Unlock()
			if err != nil {
				u.UnregisterWebSocket(c.Sock)
				log.Println("failed to send msg, cleaned up socket", err)
			}
		}
	} else { // offline
		if doStore {
			// send notification
			// unmarshal message to get sender
			msgData := MsgData{}
			err := msgpack.Unmarshal(msg, &msgData)
			if err != nil {
				log.Println("failed to unmarshal message")
				u.Pusher.Push("", []byte("Received message"))
			} else {
				marshalled, _ := json.Marshal(MsgPushNotification{
					Type: "message",
					From: base64.RawURLEncoding.EncodeToString(msgData.From),
				})
				u.Pusher.Push("", marshalled)
			}
		}
		// else message disappears silently
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

func (u *User) SetShareableData(shareableData []byte) {
	u.Mux.Lock()
	u.ShareableData = shareableData
	u.Mux.Unlock()
}

func (u *User) GetShareableData() []byte {
	u.Mux.RLock()
	defer u.Mux.RUnlock()
	return u.ShareableData
}

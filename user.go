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
	id             string
	conns          []Connection
	online         bool
	mux            *sync.RWMutex
	pusher         Pusher
	lastConnection time.Time
}

type Connection struct {
	sock *websocket.Conn
	mux  *sync.Mutex
}

type Msg struct {
	body []byte    // content
	kick time.Time // time to kick from storage
}

type MsgData struct {
	From []byte `msgpack:"f"`
}

type MsgPushNotification struct {
	Type string `json:"type"`
	From string `json:"from"`
}

func (u *User) registerWebSocket(conn *websocket.Conn) {
	c := Connection{
		sock: conn,
		mux:  new(sync.Mutex),
	}
	u.mux.Lock()
	u.conns = append(u.conns, c)
	u.online = true
	u.lastConnection = time.Now()
	u.mux.Unlock()
}

func (u *User) unregisterWebSocket(conn *websocket.Conn) {
	keep := []Connection{}
	u.mux.Lock()
	for _, c := range u.conns {
		if c.sock != conn {
			keep = append(keep, c)
		}
	}
	if len(keep) < 1 {
		u.online = false
	}
	u.conns = keep
	u.lastConnection = time.Now()
	u.mux.Unlock()
}

// Store & push message
// TODO: delegate expiry/kick to KV store
// when gobkv supports metadata & expiry.
// TODO: handle RPC errors
func (u *User) sendMessage(msg []byte, oracle *Oracle, doStore bool) {
	if doStore {
		// store it
		prefix := u.id + "/m/"
		oracle.kv.setAuto(prefix, msg)
		if !u.online {
			// add again with unread/ prefix
			// to efficiently collect later
			unreadPrefix := prefix + "unread/"
			oracle.kv.setAuto(unreadPrefix, msg)
		}
	}

	if u.online {
		for _, c := range u.conns {
			c.mux.Lock()
			err := c.sock.WriteMessage(websocket.BinaryMessage, msg)
			c.mux.Unlock()
			if err != nil {
				u.unregisterWebSocket(c.sock)
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
				u.pusher.push("", []byte("Received message"))
			} else {
				marshalled, _ := json.Marshal(MsgPushNotification{
					Type: "message",
					From: base64.RawURLEncoding.EncodeToString(msgData.From),
				})
				u.pusher.push("", marshalled)
			}
		}
		// else message disappears silently
	}
}

// Collect & send all messages, then delete expired from storage
// TODO: delegate expiry/kick to gobkv
// TODO: use buffered channel to fetch & send messages concurrently
func (u *User) sendUnread(kv *GobkvClient) {
	u.mux.RLock()
	defer u.mux.RUnlock()
	msgList, err := kv.list(u.id + "/m/unread/")
	if err != nil {
		log.Println("failed to collect msgs:", err)
		return
	}
	for _, mKey := range msgList {
		msg, err := kv.get(mKey)
		if err != nil {
			log.Println("failed to collect msg", mKey, err)
			continue
		}
		if err != nil {
			log.Println("failed to decode msg", mKey, err)
			continue
		}
		kv.del(mKey)
		for _, c := range u.conns {
			c.mux.Lock()
			err = c.sock.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				log.Println(c.sock.RemoteAddr(), "failed to send stored msg", err)
			}
			c.mux.Unlock()
		}
	}
}

func (u *User) setData(data []byte, kv *GobkvClient) error {
	return kv.set(u.id+"/data", data)
}

func (u *User) getData(kv *GobkvClient) ([]byte, error) {
	return kv.get(u.id + "/data")
}

func (u *User) setShareableData(shareableData []byte, kv *GobkvClient) {
	err := kv.set(u.id+"/shareable", shareableData)
	if err != nil {
		log.Println("failed to store shareable", err)
	}
}

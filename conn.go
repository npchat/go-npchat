package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gorilla/websocket"
	"github.com/shamaton/msgpack/v2"
)

type Response struct {
	Message  interface{} `msgpack:"message"`
	VapidKey interface{} `msgpack:"vapidKey"`
	Data     []byte      `msgpack:"data"`
	Err      interface{} `msgpack:"error"`
}

type Message struct {
	PushSub       string `msgpack:"sub"`
	Data          []byte `msgpack:"data"`
	ShareableData []byte `msgpack:"shareableData"`
}

func handleConnection(w http.ResponseWriter, r *http.Request, o *Oracle, cfg *Config) {
	idEnc := getIdFromPath(r.URL.Path)

	id, err := base64.RawURLEncoding.DecodeString(idEnc)
	if err != nil || len(id) != 32 {
		http.Error(w, fmt.Sprintf("invalid id %v", idEnc), http.StatusBadRequest)
		return
	}

	authMsg, err := getAuthMsgFromQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !verifyAuthMessage(&authMsg, id) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	u := r.Header.Get("upgrade")
	if u == "" {
		http.Error(w, "expected websocket upgrade", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	conn.SetCloseHandler(func(_ int, _ string) error {
		user, err := o.getUser(idEnc, false)
		if err != nil {
			log.Println(err, idEnc)
			return err
		}
		user.unregisterWebSocket(conn)
		return nil
	})

	user, err := o.getUser(idEnc, true)
	if err != nil {
		log.Println("failed to get user for auth", err)
		return
	}

	user.pusher.ensureKey()

	data, _ := user.getData(o.kv)
	resp := Response{
		Message:  "authed",
		VapidKey: user.pusher.publicKey,
		Data:     data,
	}
	respBin, _ := msgpack.Marshal(resp)
	err = conn.WriteMessage(websocket.BinaryMessage, respBin)
	if err != nil {
		log.Println("failed to write auth response", err)
		conn.Close()
		return
	}

	user.registerWebSocket(conn)
	user.sendUnread(o.kv)

	for {
		msgType, msgBin, err := conn.ReadMessage()
		if err != nil {
			log.Println("failed to read message", err)
			conn.Close()
			return
		}

		if msgType != websocket.BinaryMessage {
			msg, _ := msgpack.Marshal(Response{
				Message: "send only binary data serialised with msgpack",
				Err:     "invalid message type",
			})
			conn.WriteMessage(websocket.BinaryMessage, msg)
			return
		}

		var msg Message
		err = msgpack.Unmarshal(msgBin, &msg)
		if err != nil {
			log.Println("failed to unmarshal msg", err)
			return
		}

		if msg.PushSub != "" {
			log.Println("got sub", msg.PushSub)
			sub := webpush.Subscription{}
			err := json.Unmarshal([]byte(msg.PushSub), &sub)
			if err != nil {
				log.Println("failed to unmarshal push subscription")
			}
			user.pusher.addSubscription(&sub)
		}

		if msg.Data != nil && len(msg.Data) <= cfg.DataLenMax {
			user.setData(msg.Data, o.kv)
		}

		if msg.ShareableData != nil && len(msg.ShareableData) <= cfg.DataLenMax {
			user.setShareableData(msg.ShareableData, o.kv)
		}
	}
}

func checkOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin:     checkOrigin,
}

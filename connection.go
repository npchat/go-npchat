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
	Error    interface{} `msgpack:"error"`
}

type Message struct {
	PushSubscription string `msgpack:"sub"`
	Data             []byte `msgpack:"data"`
	ShareableData    []byte `msgpack:"shareableData"`
}

func HandleConnection(w http.ResponseWriter, r *http.Request, o *Oracle, opt *Options) {
	idEnc := GetIdFromPath(r.URL.Path)

	id, err := base64.RawURLEncoding.DecodeString(idEnc)
	if err != nil || len(id) != 32 {
		http.Error(w, fmt.Sprintf("invalid id %v", idEnc), http.StatusBadRequest)
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
		user, err := o.GetUser(idEnc)
		if err != nil {
			log.Println("failed to find user", idEnc)
			return err
		}
		user.UnregisterWebSocket(conn)
		return nil
	})

	for {
		authMsgType, authMsgBin, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if authMsgType != websocket.BinaryMessage {
			msg, _ := msgpack.Marshal(Response{
				Message: "send only binary data serialised with msgpack",
				Error:   "invalid message type",
			})
			conn.WriteMessage(websocket.BinaryMessage, msg)
			return
		}
		var authMsg AuthMessage
		err = msgpack.Unmarshal(authMsgBin, &authMsg)
		if err != nil {
			log.Println("failed to unmarshal auth", err)
			msg, _ := msgpack.Marshal(Response{
				Message: "failed to unmarshal message",
				Error:   err.Error(),
			})
			conn.WriteMessage(websocket.BinaryMessage, msg)
			return
		}
		if !VerifyAuthMessage(&authMsg, id) {
			msg, _ := msgpack.Marshal(Response{
				Error: "unauthorized",
			})
			conn.WriteMessage(websocket.BinaryMessage, msg)
			return
		}

		user, err := o.GetUser(idEnc)
		if err != nil {
			log.Println("failed to find user", idEnc)
			return
		}

		user.Pusher.EnsureKey()

		resp := Response{
			Message:  "authed",
			VapidKey: user.Pusher.PublicKey,
			Data:     user.GetData(),
		}
		respBin, _ := msgpack.Marshal(resp)
		conn.WriteMessage(websocket.BinaryMessage, respBin)

		user.RegisterWebSocket(conn)
		user.SendStored()

		// authed msg loop
		for {
			msgType, msgBin, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				return
			}

			if msgType != websocket.BinaryMessage {
				msg, _ := msgpack.Marshal(Response{
					Message: "send only binary data serialised with msgpack",
					Error:   "invalid message type",
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

			if msg.PushSubscription != "" {
				log.Println("got sub", msg.PushSubscription)
				sub := webpush.Subscription{}
				err := json.Unmarshal([]byte(msg.PushSubscription), &sub)
				if err != nil {
					log.Println("failed to unmarshal push subscription")
				}
				user.Pusher.AddSubscription(&sub)
			}

			if msg.Data != nil && len(msg.Data) <= opt.DataLenMax {
				user.SetData(msg.Data)
			}

			if msg.ShareableData != nil && len(msg.ShareableData) <= opt.DataLenMax {
				user.SetShareableData(msg.ShareableData)
			}
		}
	}
}

func CheckOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin:     CheckOrigin,
}

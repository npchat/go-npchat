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

type ServerResponse struct {
	Message  interface{} `json:"message"`
	VapidKey interface{} `json:"vapidKey"`
	Data     interface{} `json:"data"`
	Error    interface{} `json:"error"`
}

type DataMessage struct {
	Subscription webpush.Subscription `msgpack:"sub"`
	Data         string               `msgpack:"data"`
}

type AuthMessage struct {
	Time      []byte `msgpack:"t"`
	Sig       []byte `msgpack:"sig"`
	PublicKey []byte `msgpack:"pubKey"`
}

func HandleConnection(w http.ResponseWriter, r *http.Request, o *Oracle, opt *Options) {
	idEnc := GetIdFromPath(r.URL.Path)

	id, err := base64.RawURLEncoding.DecodeString(idEnc)
	if err != nil {
		return
	}

	if len(id) != 32 {
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
			return
		}
		var authMsg AuthMessage
		err = msgpack.Unmarshal(authMsgBin, &authMsg)
		if err != nil {
			log.Println("failed to unmarshal auth", err)
			return
		}

		// verify auth msg
		if !VerifyAuthMessage(&authMsg, id) {
			return
		}

		// user is authed
		user, err := o.GetUser(idEnc)
		if err != nil {
			log.Println("failed to find user", idEnc)
			return
		}
		user.Pusher.EnsureKey()
		r := ServerResponse{
			Message:  "handshake done",
			VapidKey: user.Pusher.PublicKey,
			Data:     user.GetData(),
		}
		rj, _ := json.Marshal(r)
		err = conn.WriteMessage(websocket.TextMessage, rj)
		if err != nil {
			log.Println(err)
			return
		}
		user.RegisterWebSocket(conn)
		user.SendStored()

		// authenticated message loop
		for {
			msgType, msgText, err := conn.ReadMessage()

			if err != nil {
				conn.Close()
				return
			}

			if msgType != websocket.TextMessage {
				log.Println("bad message", msgType, msgText)
				return
			}

			var msg DataMessage
			err = json.Unmarshal(msgText, &msg)
			if err != nil {
				log.Println("failed to unmarshal message", err)
				return
			}

			if msg.Subscription.Endpoint != "" {
				user.Pusher.AddSubscription(&msg.Subscription)
			}

			/*if msg.Get == "data" {
				resp, _ := json.Marshal(ServerResponse{
					Data: user.GetData(),
				})
				conn.WriteMessage(websocket.TextMessage, resp)
			}

			if msg.Set == "data" {
				err := user.SetData(msg.Data, opt.DataLenMax)
				if err != nil {
					log.Println("failed to set data", err)
					errResp, _ := json.Marshal(ServerResponse{
						Error: fmt.Sprintf("%v", err),
					})
					conn.WriteMessage(websocket.TextMessage, errResp)
				}
			}*/
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

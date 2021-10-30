package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func HandleRequest(mc MainChannels, w http.ResponseWriter, r *http.Request, msgTTL time.Duration) {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	idEncoded := strings.TrimLeft(r.URL.Path, "/")
	id, err := base64.RawURLEncoding.DecodeString(idEncoded)
	if err != nil {
		fmt.Println(err)
		return
	}

	// handle POST message
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println("Error reading body ", err)
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		mc.Msg <- ChatMessage{
			Id: idEncoded,
			Msg: StorableMessage{
				Body: body,
				Time: time.Now().Add(msgTTL),
			},
		}
		resp := ServerMessage{Message: "sent"}
		rj, err := json.Marshal(resp)
		if err != nil {
			fmt.Println("failed to marshal json", err)
		}
		w.Write(rj)
		return
	}

	ugh := r.Header.Get("upgrade")
	if ugh == "" {
		w.Write([]byte("Expected websocket upgrade"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
	}

	conn.SetCloseHandler(func(_ int, _ string) error {
		mc.Unregister <- Session{Id: idEncoded, Conn: conn}
		return nil
	})

	for {
		msgType, msgTxt, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			conn.Close()
			return
		}
		if msgType != websocket.TextMessage {
			fmt.Println("send only json")
			conn.Close()
			return
		}
		var msg ClientMessage
		err = json.Unmarshal(msgTxt, &msg)
		if err != nil {
			fmt.Println(err)
			conn.Close()
			return
		}
		if AuthenticateSocket(conn, &msg, mc.ChallengeCount, mc.PrivKey, id) {
			fmt.Println(r.RemoteAddr, "authed")
			mc.Register <- Session{Id: idEncoded, Conn: conn}
		}
	}
}

func CheckOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     CheckOrigin,
}

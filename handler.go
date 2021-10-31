package main

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
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
		log.Println(err)
		return
	}

	// handle POST message
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		r.Body.Close()
		mc.Msg <- ChatMessage{
			Id: idEncoded,
			Msg: StorableMessage{
				Body: body,
				Time: time.Now().Add(msgTTL),
			},
		}
		resp := ServerMessage{Message: "sent"}
		rj, _ := json.Marshal(resp)
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
		log.Println(err)
	}

	conn.SetCloseHandler(func(_ int, _ string) error {
		mc.Unregister <- Session{Id: idEncoded, Conn: conn}
		return nil
	})

	for {
		msgType, msgTxt, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			return
		}
		if msgType != websocket.TextMessage {
			conn.Close()
			return
		}
		var msg ClientMessage
		err = json.Unmarshal(msgTxt, &msg)
		if err != nil {
			conn.Close()
			return
		}
		if msg.Get == "challenge" {
			mc.ChallengeCount <- 1
			privKey := <-mc.PrivKey
			HandleChallengeRequest(conn, &privKey)
		} else if msg.Solution != "" {
			mc.ChallengeCount <- 0  // don't increment counter
			privKey := <-mc.PrivKey // just get key
			if !VerifySolution(&msg, id, &privKey.PublicKey) {
				conn.Close()
				return
			} else {
				r := ServerMessage{Message: "handshake done"}
				rj, _ := json.Marshal(r)
				err := conn.WriteMessage(websocket.TextMessage, rj)
				if err != nil {
					log.Println(err)
					conn.Close()
					return
				}
				mc.Register <- Session{Id: idEncoded, Conn: conn}
				log.Println(idEncoded, "<- registered")
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

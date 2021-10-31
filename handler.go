package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func GetIdFromPath(path string) string {
	return strings.TrimLeft(path, "/")
}

func HandlePostRequest(w http.ResponseWriter, r *http.Request,
	msgTTL time.Duration, isActive bool,
	recvChan chan Message, store chan MessageWithId) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	msg := Message{ // Get ID and send on corresponding chan
		Body: body,
		Time: time.Now().Add(msgTTL),
	}
	if isActive {
		recvChan <- msg
	} else {
		store <- MessageWithId{
			Id:      GetIdFromPath(r.URL.Path),
			Message: msg,
		}
	}
	resp := ServerResponse{Message: "sent"}
	rj, _ := json.Marshal(resp)
	w.Write(rj)
}

func HandleConnectionRequest(w http.ResponseWriter, r *http.Request,
	register chan Registration, unregister chan string, ask chan string, retrv chan []Message) {

	idEncoded := GetIdFromPath(r.URL.Path)
	id, err := base64.RawURLEncoding.DecodeString(idEncoded)
	if err != nil {
		log.Println(err)
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
		return
	}
	defer conn.Close()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Println("keygen failed", err)
	}

	for {
		msgType, msgTxt, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.TextMessage {
			return
		}
		var msg ClientMessage
		err = json.Unmarshal(msgTxt, &msg)
		if err != nil {
			return
		}
		if msg.Get == "challenge" {
			challenge, err := GetChallenge(conn, privKey)
			if err != nil {
				log.Println(err)
				return
			}
			buf := new(bytes.Buffer)
			json.NewEncoder(buf).Encode(challenge)
			err = conn.WriteMessage(websocket.TextMessage, buf.Bytes())
			if err != nil {
				return
			}
		} else if msg.Solution != "" {
			if !VerifySolution(&msg, id, &privKey.PublicKey) {
				return
			} else {
				r := ServerResponse{Message: "handshake done"}
				rj, _ := json.Marshal(r)
				err := conn.WriteMessage(websocket.TextMessage, rj)
				if err != nil {
					log.Println(err)
					return
				}
				// register
				recv := make(chan Message)
				register <- Registration{
					Id:       idEncoded,
					RecvChan: recv,
				}
				HandleSession(idEncoded, conn, recv, ask, retrv, unregister)
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

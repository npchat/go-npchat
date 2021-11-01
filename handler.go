package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type ServerResponse struct {
	Message string `json:"message"`
}

type ClientMessage struct {
	Get       string    `json:"get"`
	Challenge Challenge `json:"challenge"`
	PublicKey string    `json:"publicKey"`
	Solution  string    `json:"solution"`
}

func GetIdFromPath(path string) string {
	return strings.TrimLeft(path, "/")
}

func HandlePostRequest(w http.ResponseWriter, r *http.Request, ss *SessionStore, ms *MessageStore, opt *Options) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	msg := Message{
		Body: body,
		Time: time.Now().Add(opt.MessageTTL),
	}
	id := GetIdFromPath(r.URL.Path)
	ss.Mtx.RLock()
	isActive := ss.Active[id]
	recv := ss.Recv[id]
	ss.Mtx.RUnlock()
	if isActive {
		recv <- &msg
	} else {
		ms.Store <- MessageWithId{
			Id:      id,
			Message: msg,
		}
	}
	resp := ServerResponse{Message: "sent"}
	rj, _ := json.Marshal(resp)
	w.Write(rj)
}

func HandleConnectionRequest(w http.ResponseWriter, r *http.Request, ss *SessionStore, ms *MessageStore) {
	idEnc := GetIdFromPath(r.URL.Path)
	id, err := base64.RawURLEncoding.DecodeString(idEnc)
	if err != nil {
		log.Println(err)
		return
	}

	u := r.Header.Get("upgrade")
	if u == "" {
		w.Write([]byte("Expected websocket upgrade"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	privKey := make(chan *ecdsa.PrivateKey)
	challengeCount := make(chan int)
	go KeepFreshKey(privKey, challengeCount, 20)

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
			challengeCount <- 1
			priv := <-privKey
			challenge, err := GetChallenge(priv)
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
			challengeCount <- 0
			priv := <-privKey
			if !VerifySolution(&msg, id, &priv.PublicKey) {
				return
			} else {
				r := ServerResponse{Message: "handshake done"}
				rj, _ := json.Marshal(r)
				err := conn.WriteMessage(websocket.TextMessage, rj)
				if err != nil {
					log.Println(err)
					return
				}
				go HandleSession(idEnc, conn, ss, ms)
			}
		}
	}
}

func KeepFreshKey(privKey chan *ecdsa.PrivateKey, challengeCount chan int, limit int) {
	count := 0
	curKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	for {
		count += <-challengeCount
		if count >= limit {
			curKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			count = 0
		}
		privKey <- curKey
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

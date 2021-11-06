package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type ServerResponse struct {
	Message  string `json:"message"`
	VapidKey string `json:"vapidKey"`
}

type ClientMessage struct {
	Get          string    `json:"get"`
	Challenge    Challenge `json:"challenge"`
	PublicKey    string    `json:"publicKey"`
	Solution     string    `json:"solution"`
	Subscription string    `json:"subscription"`
}

func HandleConnection(w http.ResponseWriter, r *http.Request, o *Oracle) {
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

	conn.SetCloseHandler(func(_ int, _ string) error {
		o.GetUser(idEnc).UnregisterWebSocket(conn)
		return nil
	})

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
		}
		if msg.Solution != "" {
			challengeCount <- 0
			priv := <-privKey
			if !VerifySolution(&msg, id, &priv.PublicKey) {
				return
			} else {
				user := o.GetUser(idEnc)
				user.Pusher.EnsureKey()
				r := ServerResponse{Message: "handshake done", VapidKey: user.Pusher.PublicKey}
				rj, _ := json.Marshal(r)
				err := conn.WriteMessage(websocket.TextMessage, rj)
				if err != nil {
					log.Println(err)
					return
				}
				user.RegisterWebSocket(conn)
				user.SendStored()
				_, subMsg, _ := conn.ReadMessage()
				if string(subMsg) != "" {
					log.Println("got subscription")
					user.Pusher.AddSubscription(subMsg)
				}
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

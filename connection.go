package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type ServerResponse struct {
	Message  interface{} `json:"message"`
	VapidKey interface{} `json:"vapidKey"`
	Data     interface{} `json:"data"`
	Error    interface{} `json:"error"`
}

type ClientMessage struct {
	Get          string    `json:"get"`
	Set          string    `json:"set"`
	Challenge    Challenge `json:"challenge"`
	PublicKey    string    `json:"publicKey"`
	Solution     string    `json:"solution"`
	Subscription []byte    `json:"subscription"`
	Data         string    `json:"data"`
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

					var msg ClientMessage
					err = json.Unmarshal(msgText, &msg)
					if err != nil {
						log.Println("failed to unmarshal message", err)
						return
					}

					if string(msg.Subscription) != "" {
						log.Println("got subscription")
						user.Pusher.AddSubscription(msg.Subscription)
					}

					if msg.Get == "data" {
						log.Println("got request for data")
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
					}
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

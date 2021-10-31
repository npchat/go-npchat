package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// Refresh key after given limit for challenge count
func KeepFreshKey(challCount chan int, priv chan ecdsa.PrivateKey, limit int) {
	count := 0
	curKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println(err)
		return
	}
	for c := range challCount {
		count += c
		if count >= limit {
			curKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				fmt.Println(err)
				return
			}
			count = 0
		}
		priv <- *curKey
	}
}

func PumpMessages(mc MainChannels, hc HousekeepingChannels) {
	active := make(map[*websocket.Conn]bool)
	sessions := make(map[string]Session)
	store := make(map[string][]StorableMessage)

	for {
		select {
		case s := <-mc.Register: // session
			sessions[s.Id] = s
			active[s.Conn] = true
			m := store[s.Id]
			delete(store, s.Id)
			SessionStart(s.Conn, s.Id, m, mc.Msg)
		case s := <-mc.Unregister: // session
			active[s.Conn] = false
		case m := <-mc.Msg: // serve or store
			s := sessions[m.Id]
			if active[s.Conn] {
				err := s.Conn.WriteMessage(websocket.TextMessage, m.Msg.Body)
				if err != nil {
					k := []StorableMessage{}
					k = append(k, store[m.Id]...)
					k = append(k, m.Msg)
					store[m.Id] = k
				}
			} else {
				k := []StorableMessage{}
				k = append(k, store[m.Id]...)
				k = append(k, m.Msg)
				store[m.Id] = k
			}
		case <-hc.GetKeys:
			keys := make([]string, 0, len(store))
			for k := range store {
				if len(store[k]) < 1 {
					delete(store, k)
				} else {
					keys = append(keys, k)
				}
			}
			hc.Keys <- keys
		case i := <-hc.GetMsgsForKey:
			for _, m := range store[i] {
				hc.MsgsForKey <- m
			}
			hc.MsgsForKey <- StorableMessage{}
		case skv := <-hc.StoreKeyValue:
			store[skv.Id] = skv.Msgs
		}
	}
}

// Called when a session is registered
func SessionStart(conn *websocket.Conn, id string, stored []StorableMessage, store chan ChatMessage) {
	r := ServerMessage{Message: "handshake done"}
	rj, _ := json.Marshal(r)
	err := conn.WriteMessage(websocket.TextMessage, rj)
	if err != nil {
		fmt.Println(err)
		conn.Close()
		return
	}
	for _, mS := range stored {
		err := conn.WriteMessage(websocket.TextMessage, mS.Body)
		if err != nil {
			fmt.Println(err)
			// push it back to storage
			store <- ChatMessage{Id: id, Msg: mS}
		}
	}
}

func main() {
	opt := GetOptions()
	fmt.Println(opt)

	mc := MainChannels{
		ChallengeCount: make(chan int),
		PrivKey:        make(chan ecdsa.PrivateKey),
		Msg:            make(chan ChatMessage),
		Register:       make(chan Session),
		Unregister:     make(chan Session),
	}

	hc := HousekeepingChannels{
		GetKeys:       make(chan bool),
		Keys:          make(chan []string),
		GetMsgsForKey: make(chan string),
		MsgsForKey:    make(chan StorableMessage),
		StoreKeyValue: make(chan StoreKeyValue),
	}

	go KeepFreshKey(mc.ChallengeCount, mc.PrivKey, opt.FreshKey)

	go PumpMessages(mc, hc)

	go CleanStore(hc, opt.CleanPeriod)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		HandleRequest(mc, w, r, opt.MessageTTL)
	})

	addr := fmt.Sprintf(":%v", opt.Port)
	if opt.CertFile != "" && opt.PrivKeyFile != "" {
		fmt.Printf("listening on %v, serving with TLS\n", addr)
		err := http.ListenAndServeTLS(addr, opt.CertFile, opt.PrivKeyFile, nil)
		if err != nil {
			fmt.Println("failed to start HTTPS server\n", err)
		}
	} else {
		fmt.Printf("listening on %v\n", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Println("failed to start HTTP server\n", err)
		}
	}
}

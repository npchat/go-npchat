package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

const PORT = 8000
const DBFILE = "msg.log"

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

func GetStoredMessages(id string, messages chan []byte) {
	// return messages where Id prefix matches
	f, err := os.OpenFile(DBFILE, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ln := sc.Text()
		if strings.HasPrefix(ln, id) {
			fmt.Println(ln)
			m := []byte(ln)[len(id):]
			messages <- m
		}
	}
	close(messages)
	f.Close()
}

func PostAuth(conn *websocket.Conn, connMap map[string]chan *websocket.Conn, id string) {
	r := ServerMessage{Message: "handshake done"}
	rj, _ := json.Marshal(r)
	err := conn.WriteMessage(websocket.TextMessage, rj)
	if err != nil {
		fmt.Println(err)
		conn.Close()
		return
	}
	sm := make(chan []byte)
	go GetStoredMessages(id, sm)
	for m := range sm {
		err := conn.WriteMessage(websocket.TextMessage, m)
		if err != nil {
			fmt.Println(err)
		}
	}
	// register this conn
	if connMap[id] == nil {
		connMap[id] = make(chan *websocket.Conn)
	}
	wsChan := connMap[id]
	for {
		select <- done:
		break
		case: wsChan <- conn
			fmt.Println("pushed conn")
	}
}

func main() {
	WriteToStore("Started", "")
	opt := GetOptionsFromFlags()
	fmt.Println(opt)

	challCount := make(chan int)
	defer close(challCount)

	priv := make(chan ecdsa.PrivateKey)
	defer close(priv)

	// map of channels for websockets
	connMap := make(map[string]chan *websocket.Conn)

	go KeepFreshKey(challCount, priv, 5)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

			wsChan := connMap[idEncoded]
			select {
			case ws := <-wsChan:
				if ws != nil {
					err = ws.WriteMessage(websocket.TextMessage, body)
					if err != nil {
						fmt.Println("failed sending via websocket", err)
					} else {
						fmt.Println(r.RemoteAddr, "sent using websocket")
						break
					}
				}
				WriteToStore("", idEncoded+string(body))
				fmt.Println(r.RemoteAddr, "stored")
				break
			default:
				WriteToStore("", idEncoded+string(body))
				fmt.Println(r.RemoteAddr, "stored")
			}

			resp := ServerMessage{Message: "sent"}
			rj, err := json.Marshal(resp)
			if err != nil {
				fmt.Println("failed to marshal json", err)
			}
			w.Write(rj)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
		}
		isAuthed := false
		for {
			msgType, msgTxt, err := conn.ReadMessage()
			if err != nil {
				fmt.Println(err)
				break
			}
			if msgType != websocket.TextMessage {
				fmt.Println("send only json")
				break
			}
			var msg ClientMessage
			err = json.Unmarshal(msgTxt, &msg)
			if err != nil {
				fmt.Println(err)
				break
			}
			isAuthed = AuthenticateSocket(conn, &msg, challCount, priv, id)
			if isAuthed {
				fmt.Println(r.RemoteAddr, "authed")
				break
			}
		}
		fmt.Println(r.RemoteAddr, "isAuthed", isAuthed)
		if isAuthed {
			PostAuth(conn, connMap, idEncoded)
		}
		fmt.Println(r.RemoteAddr, "done, closing")
		err = conn.Close()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(r.RemoteAddr, "closed")
	})

	addr := fmt.Sprintf(":%v", opt.Port)
	if opt.CertFile != "" && opt.KeyFile != "" {
		fmt.Printf("listening on %v, serving with TLS\n", addr)
		err := http.ListenAndServeTLS(addr, opt.CertFile, opt.KeyFile, nil)
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

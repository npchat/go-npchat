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

func PumpMessages(msg chan ChatMessage, register chan Session, unregister chan Session) {
	active := make(map[*websocket.Conn]bool)
	sessions := make(map[string]Session)
	for {
		select {
		case s := <-register:
			sessions[s.Id] = s
			active[s.Conn] = true
			PostAuth(s.Conn, s.Id)
		case s := <-unregister:
			active[s.Conn] = false
			fmt.Println(s.Conn.RemoteAddr(), s.Id, "closed")
		case m := <-msg:
			// find session by Id
			s := sessions[m.Id]
			if active[s.Conn] {
				err := s.Conn.WriteMessage(websocket.TextMessage, m.Body)
				if err != nil {
					fmt.Println(s.Conn.RemoteAddr(), m.Id, "<- ws closed?!")
					WriteToStore("", m.Id+string(m.Body))
					fmt.Println(m.Id, "<- stored")
				} else {
					fmt.Println(s.Conn.RemoteAddr(), m.Id, "<- sent")
				}
			} else {
				WriteToStore("", m.Id+string(m.Body))
				fmt.Println(m.Id, "<- stored")
			}
		}
	}
}

func PostAuth(conn *websocket.Conn, id string) {
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
}

func main() {
	WriteToStore("Started", "")
	opt := GetOptionsFromFlags()
	fmt.Println(opt)

	challCount := make(chan int)
	defer close(challCount)

	priv := make(chan ecdsa.PrivateKey)
	defer close(priv)

	msg := make(chan ChatMessage)
	register := make(chan Session)
	unregister := make(chan Session)

	go KeepFreshKey(challCount, priv, 5)

	go PumpMessages(msg, register, unregister)

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
			msg <- ChatMessage{Id: idEncoded, Body: body}
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

		conn.SetCloseHandler(func(code int, text string) error {
			fmt.Println("JOEY!", code, text)
			unregister <- Session{Id: idEncoded, Conn: conn}
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
			if AuthenticateSocket(conn, &msg, challCount, priv, id) {
				fmt.Println(r.RemoteAddr, "authed")
				register <- Session{Id: idEncoded, Conn: conn}
			}
		}
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

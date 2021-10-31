package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Sessions struct {
	Active     map[string]bool
	Recv       map[string]chan Message
	Register   chan Registration
	Unregister chan string
	Mtx        sync.Mutex
}

func (s *Sessions) Manage() {
	for {
		select {
		case r := <-s.Register:
			s.Mtx.Lock()
			s.Recv[r.Id] = r.RecvChan
			s.Active[r.Id] = true
			s.Mtx.Unlock()
		case u := <-s.Unregister:
			s.Mtx.Lock()
			delete(s.Active, u)
			delete(s.Recv, u)
			s.Mtx.Unlock()
		}
	}
}

func HandleSession(id string, conn *websocket.Conn,
	recv chan Message, ask chan string, retrv chan []Message, unregister chan string) {
	done := make(chan bool)
	conn.SetCloseHandler(func(_ int, _ string) error {
		unregister <- id
		conn.WriteControl(websocket.CloseGoingAway, nil, time.Now().Add(time.Second))
		conn.Close()
		done <- true
		return nil
	})
	// serve stored messages
	ask <- id
	stored := <-retrv
	for _, m := range stored {
		err := conn.WriteMessage(websocket.TextMessage, m.Body)
		if err != nil {
			log.Println("failed sending stored message", err)
		}
	}
	// serve incoming messages
	go func() {
		for m := range recv {
			conn.WriteMessage(websocket.TextMessage, m.Body)
		}
	}()
	// keep reading to detect close message
	go func() {
		conn.ReadMessage()
	}()
	<-done
}

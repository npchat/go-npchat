package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type SessionStore struct {
	Active     map[string]bool
	Recv       map[string]chan *Message
	Register   chan Registration
	Unregister chan string
	Mtx        sync.RWMutex
}

type Registration struct {
	Id       string
	RecvChan chan *Message
}

func (s *SessionStore) Manage() {
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

func HandleSession(id string, conn *websocket.Conn, ss *SessionStore, ms *MessageStore) {
	recv := make(chan *Message)
	ss.Register <- Registration{
		Id:       id,
		RecvChan: recv,
	}
	mtx := new(sync.Mutex)
	done := make(chan bool)
	conn.SetCloseHandler(func(_ int, _ string) error {
		ss.Unregister <- id
		conn.WriteControl(websocket.CloseGoingAway, nil, time.Now().Add(time.Second))
		conn.Close()
		done <- true
		return nil
	})
	// serve stored messages
	ms.Ask <- id
	stored := <-ms.Retrv
	for _, m := range stored {
		mtx.Lock()
		err := conn.WriteMessage(websocket.TextMessage, m.Body)
		mtx.Unlock()
		if err != nil {
			log.Println("failed sending stored message", err)
		}
	}
	// serve incoming messages
	go func() {
		for m := range recv {
			mtx.Lock()
			conn.WriteMessage(websocket.TextMessage, m.Body)
			mtx.Unlock()
		}
	}()
	// ping client to ensure connection is up
	// this prevents issues from improperly closed connections
	go func() {
		for {
			mtx.Lock()
			err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			mtx.Unlock()
			if err != nil {
				ss.Unregister <- id
				conn.Close()
				done <- true
			}
			time.Sleep(time.Second * 30)
		}
	}()
	<-done
}

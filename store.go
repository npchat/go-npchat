package main

import (
	"sync"
	"time"
)

type MessageStore struct {
	M           map[string][]Message
	Mtx         sync.RWMutex
	Store       chan MessageWithId
	Ask         chan string
	Retrv       chan []Message
	CleanPeriod time.Duration
}

type MessageWithId struct {
	Id      string
	Message Message
}

type Message struct {
	Body []byte
	Time time.Time
}

func (ms *MessageStore) Manage() {
	for {
		select {
		case msg := <-ms.Store:
			k := []Message{}
			ms.Mtx.RLock()
			k = append(k, ms.M[msg.Id]...)
			ms.Mtx.RUnlock()
			k = append(k, msg.Message)
			ms.Mtx.Lock()
			ms.M[msg.Id] = k
			ms.Mtx.Unlock()
		case a := <-ms.Ask:
			ms.Mtx.RLock()
			ms.Retrv <- ms.M[a]
			ms.Mtx.RUnlock()
		}
	}
}

func (ms *MessageStore) KeepClean() {
	for {
		ms.Mtx.Lock()
		for k := range ms.M {
			keep := []Message{}
			for _, m := range ms.M[k] {
				if m.Time.Before(time.Now()) {
					keep = append(keep, m)
				}
			}
			if len(keep) < 1 {
				delete(ms.M, k)
			} else {
				ms.M[k] = keep
			}
		}
		ms.Mtx.Unlock()
		time.Sleep(ms.CleanPeriod)
	}
}

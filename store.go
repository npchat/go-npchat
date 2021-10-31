package main

import (
	"sync"
	"time"
)

type MessageStore struct {
	M           map[string][]Message
	Mtx         sync.Mutex
	Store       chan MessageWithId
	Ask         chan string
	Retrv       chan []Message
	CleanPeriod time.Duration
}

func (ms *MessageStore) Manage() {
	for {
		select {
		case msg := <-ms.Store:
			ms.Mtx.Lock()
			k := []Message{}
			k = append(k, ms.M[msg.Id]...)
			k = append(k, msg.Message)
			ms.M[msg.Id] = k
			ms.Mtx.Unlock()
		case a := <-ms.Ask:
			ms.Mtx.Lock()
			ms.Retrv <- ms.M[a]
			ms.Mtx.Unlock()
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

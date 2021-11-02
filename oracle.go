package main

import (
	"sync"
	"time"
)

type Oracle struct {
	Users       map[string]*User
	Mux         *sync.RWMutex
	CleanPeriod time.Duration
	MsgTTL      time.Duration
}

func (o *Oracle) GetUser(id string) *User {
	o.Mux.RLock()
	u := o.Users[id]
	o.Mux.RUnlock()
	if u != nil {
		return u
	} else {
		// make one
		o.Mux.Lock()
		o.Users[id] = &User{
			Msgs:  make([]Msg, 0),
			Conns: make([]Connection, 0),
			Mux:   new(sync.RWMutex),
		}
		defer o.Mux.Unlock()
		return o.Users[id]
	}
}

func (o *Oracle) KeepClean() {
	for {
		o.Mux.Lock()
		for id, u := range o.Users {
			keep := []Msg{}
			for _, m := range u.Msgs {
				if time.Now().Before(m.Kick) {
					keep = append(keep, m)
				}
			}
			if len(keep) < 1 && !u.Online {
				delete(o.Users, id)
			}
			u.Msgs = keep
		}
		o.Mux.Unlock()
		time.Sleep(o.CleanPeriod)
	}
}

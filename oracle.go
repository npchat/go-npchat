package main

import (
	"encoding/base64"
	"errors"
	"log"
	"sync"
	"time"
)

type Oracle struct {
	Users       map[string]*User
	Mux         *sync.RWMutex `json:"-"`
	CleanPeriod time.Duration `json:"-"`
	MsgTTL      time.Duration `json:"-"`
	PersistFile string        `json:"-"`
}

func (o *Oracle) GetUser(id string) (*User, error) {
	o.Mux.RLock()
	u := o.Users[id]
	o.Mux.RUnlock()
	if u != nil {
		if u.Mux == nil {
			u.Mux = new(sync.RWMutex)
		}
		return u, nil
	} else {
		// validate id
		idBytes, err := base64.RawURLEncoding.DecodeString(id)
		if err != nil {
			return nil, err
		}
		if len(idBytes) != 32 {
			return nil, errors.New("invalid id")
		}
		// make one
		o.Mux.Lock()
		o.Users[id] = &User{
			Msgs:  make([]Msg, 0),
			Conns: make([]Connection, 0),
			Mux:   new(sync.RWMutex),
		}
		defer o.Mux.Unlock()
		return o.Users[id], nil
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
			// remove user if:
			// has no messages
			// is offline
			// has no push subscription
			// has no stored data
			if len(keep) < 1 && !u.Online && u.Pusher.Subscription == nil && u.Data == "" {
				delete(o.Users, id)
			}
			u.Msgs = keep
		}
		err := o.PersistState()
		if err != nil {
			log.Println("failed to persist state", err)
		}
		o.Mux.Unlock()
		time.Sleep(o.CleanPeriod)
	}
}

func (o *Oracle) PersistState() error {
	if o.PersistFile != "" {
		return Save(o.PersistFile, o)
	}
	return nil
}

func (o *Oracle) LoadState() error {
	if o.PersistFile != "" {
		return Load(o.PersistFile, o)
	}
	return nil
}

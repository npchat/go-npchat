package main

import (
	"encoding/base64"
	"errors"
	"log"
	"sync"
	"time"
)

type Oracle struct {
	Users   map[string]*User
	Mux     *sync.RWMutex `json:"-"`
	Options *Options      `json:"-"`
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
		if err != nil || len(idBytes) != 32 {
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
			// is offline && (
			// last connection older than UserTTL ||
			// has no messages &&
			// has no push subscription &&
			// has no stored data &&
			// has no shareable data )
			if !u.Online && (time.Now().After(u.LastConnection.Add(o.Options.UserTTL)) ||
				len(keep) < 1 && u.Pusher.Subscription == nil && u.Data == nil && u.ShareableData == nil) {
				delete(o.Users, id)
				log.Println("cleaned up", id)
			} else {
				u.Msgs = keep
			}
		}
		err := o.WriteState()
		if err != nil {
			log.Println("failed to write state", err)
		}
		o.Mux.Unlock()
		time.Sleep(o.Options.CleanPeriod)
	}
}

func (o *Oracle) WriteState() error {
	if o.Options.PersistFile != "" {
		return Write(o.Options.PersistFile, o)
	}
	return nil
}

func (o *Oracle) ReadState() error {
	if o.Options.PersistFile != "" {
		return Read(o.Options.PersistFile, o)
	}
	return nil
}

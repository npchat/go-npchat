package main

import (
	"encoding/base64"
	"errors"
	"log"
	"sync"
	"time"
)

type Oracle struct {
	users  map[string]*User
	mux    *sync.RWMutex
	config *Config
	kv     *GobkvClient
}

func (o *Oracle) getUser(id string, makeIfNotFound bool) (*User, error) {
	o.mux.RLock()
	u := o.users[id]
	o.mux.RUnlock()
	if u != nil {
		if u.mux == nil {
			u.mux = new(sync.RWMutex)
		}
		return u, nil
	} else if makeIfNotFound {
		// validate id
		idBytes, err := base64.RawURLEncoding.DecodeString(id)
		if err != nil || len(idBytes) != 32 {
			return nil, errors.New("invalid id")
		}
		// make one
		o.mux.Lock()
		o.users[id] = &User{
			id:    id,
			conns: make([]Connection, 0),
			mux:   new(sync.RWMutex),
		}
		defer o.mux.Unlock()
		return o.users[id], nil
	}
	return nil, errors.New("no user found")
}

func (o *Oracle) keepClean() {
	for {
		o.mux.Lock()
		for id, u := range o.users {
			// remove user if:
			// is offline && (
			// last connection older than UserTTL ||
			// has no messages &&
			// has no push subscription &&
			// has no stored data &&
			// has no shareable data )
			expires := u.lastConnection.Add(o.config.UserTTL.Duration)
			hasExpired := time.Now().After(expires)
			if !u.online && hasExpired {
				delete(o.users, id)
				log.Println("cleaned up", id)
			}
		}
		o.mux.Unlock()
		time.Sleep(o.config.CleanPeriod.Duration)
	}
}

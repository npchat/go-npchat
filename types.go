package main

import (
	"crypto/ecdsa"
	"time"

	"github.com/gorilla/websocket"
)

type MainChannels struct {
	ChallengeCount chan int
	PrivKey        chan ecdsa.PrivateKey
	Msg            chan ChatMessage
	Register       chan Session
	Unregister     chan Session
}

type HousekeepingChannels struct {
	GetKeys       chan bool
	Keys          chan []string
	GetMsgsForKey chan string
	MsgsForKey    chan StorableMessage
	StoreKeyValue chan StoreKeyValue
}

type StoreKeyValue struct {
	Id   string
	Msgs []StorableMessage
}

type ServerMessage struct {
	Message string `json:"message"`
}

type ServerChallenge struct {
	Challenge Challenge `json:"challenge"`
}

type ClientMessage struct {
	Get       string    `json:"get"`
	Challenge Challenge `json:"challenge"`
	PublicKey string    `json:"publicKey"`
	Solution  string    `json:"solution"`
}

type ChatMessage struct {
	Id  string
	Msg StorableMessage
}

type StorableMessage struct {
	Body []byte
	Time time.Time
}

type Session struct {
	Id   string
	Conn *websocket.Conn
}

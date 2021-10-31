package main

import (
	"time"
)

type ServerResponse struct {
	Message string `json:"message"`
}

type Challenge struct {
	Txt string `json:"txt"`
	Sig string `json:"sig"`
}

type ChallengeWrapper struct {
	Challenge Challenge `json:"challenge"`
}

type ClientMessage struct {
	Get       string    `json:"get"`
	Challenge Challenge `json:"challenge"`
	PublicKey string    `json:"publicKey"`
	Solution  string    `json:"solution"`
}

type MessageWithId struct {
	Id      string
	Message Message
}

type Message struct {
	Body []byte
	Time time.Time
}

type Registration struct {
	Id       string
	RecvChan chan Message
}

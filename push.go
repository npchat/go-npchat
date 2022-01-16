package main

import (
	"log"
	"time"

	"github.com/SherClockHolmes/webpush-go"
)

type Pusher struct {
	Subscription *webpush.Subscription
	PrivateKey   string
	PublicKey    string
	Last         time.Time
}

func (p *Pusher) GenerateKeys() {
	var err error
	p.PrivateKey, p.PublicKey, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Println("failed to generate VAPID keys", err)
	} else {
		log.Println("got new VAPID keys", p.PublicKey)
	}
}

func (p *Pusher) EnsureKey() {
	if p.PublicKey == "" {
		p.GenerateKeys()
	}
}

func (p *Pusher) AddSubscription(subscription *webpush.Subscription) {
	p.Subscription = subscription
	p.Last = time.Now()
}

func (p *Pusher) Push(subscriber string, message []byte) {
	if p.Subscription == nil {
		return
	}
	if time.Now().Before(p.Last.Add(time.Minute)) {
		return
	}
	resp, err := webpush.SendNotification(message, p.Subscription, &webpush.Options{
		Subscriber:      subscriber,
		VAPIDPublicKey:  p.PublicKey,
		VAPIDPrivateKey: p.PrivateKey,
		TTL:             120,
	})
	if err != nil {
		log.Println("failed to send push notification", err)
	}
	resp.Body.Close()
	p.Last = time.Now()
}

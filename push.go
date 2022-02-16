package main

import (
	"log"
	"time"

	"github.com/SherClockHolmes/webpush-go"
)

type Pusher struct {
	sub        *webpush.Subscription
	privateKey string
	publicKey  string
	last       time.Time
}

func (p *Pusher) generateKeys() {
	var err error
	p.privateKey, p.publicKey, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Println("failed to generate VAPID keys", err)
	} else {
		log.Println("got new VAPID keys", p.publicKey)
	}
}

func (p *Pusher) ensureKey() {
	if p.publicKey == "" {
		p.generateKeys()
	}
}

func (p *Pusher) addSubscription(subscription *webpush.Subscription) {
	p.sub = subscription
	p.last = time.Now()
}

func (p *Pusher) push(subscriber string, message []byte) {
	if p.sub == nil {
		return
	}
	if time.Now().Before(p.last.Add(time.Minute)) {
		return
	}
	resp, err := webpush.SendNotification(message, p.sub, &webpush.Options{
		Subscriber:      subscriber,
		VAPIDPublicKey:  p.publicKey,
		VAPIDPrivateKey: p.privateKey,
		TTL:             120,
	})
	if err != nil {
		log.Println("failed to send push notification", err)
	}
	resp.Body.Close()
	p.last = time.Now()
}

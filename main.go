package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	opt := GetOptions()
	log.Printf("%+v'\n", opt)

	ms := MessageStore{
		M:           make(map[string][]Message),
		Store:       make(chan MessageWithId),
		Ask:         make(chan string),
		Retrv:       make(chan []Message),
		CleanPeriod: opt.CleanPeriod,
	}

	go ms.Manage()
	go ms.KeepClean()

	sessions := Sessions{
		Active:     make(map[string]bool),
		Recv:       make(map[string]chan Message),
		Register:   make(chan Registration),
		Unregister: make(chan string),
	}

	go sessions.Manage()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if r.Method == "POST" {
			id := GetIdFromPath(r.URL.Path)
			sessions.Mtx.Lock()
			isActive := sessions.Active[id]
			recvChan := sessions.Recv[id]
			sessions.Mtx.Unlock()
			HandlePostRequest(w, r, opt.MessageTTL, isActive, recvChan, ms.Store)
		} else {
			HandleConnectionRequest(w, r, sessions.Register, sessions.Unregister, ms.Ask, ms.Retrv)
		}
	})

	addr := fmt.Sprintf(":%v", opt.Port)
	if opt.CertFile != "" && opt.PrivKeyFile != "" {
		log.Printf("listening on %v, serving with TLS\n", addr)
		err := http.ListenAndServeTLS(addr, opt.CertFile, opt.PrivKeyFile, nil)
		if err != nil {
			log.Println("failed to start HTTPS server\n", err)
		}
	} else {
		log.Printf("listening on %v\n", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Println("failed to start HTTP server\n", err)
		}
	}
}

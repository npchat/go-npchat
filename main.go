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

	ss := SessionStore{
		Active:     make(map[string]bool),
		Recv:       make(map[string]chan *Message),
		Register:   make(chan Registration),
		Unregister: make(chan string),
	}

	go ss.Manage()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if r.Method == "POST" {
			HandlePostRequest(w, r, &ss, &ms, &opt)
		} else {
			HandleConnectionRequest(w, r, &ss, &ms)
		}
	})

	addr := fmt.Sprintf(":%v", opt.Port)
	if opt.CertFile != "" && opt.PrivKeyFile != "" {
		log.Printf("listening on %v, serving with TLS\n", addr)
		err := http.ListenAndServeTLS(addr, opt.CertFile, opt.PrivKeyFile, nil)
		if err != nil {
			log.Println("failed to start HTTPS server", err)
		}
	} else {
		log.Printf("listening on %v\n", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Println("failed to start HTTP server", err)
		}
	}
}

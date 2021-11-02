package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {
	opt := GetOptions()
	log.Printf("%+v'\n", opt)

	oracle := Oracle{
		Users:       make(map[string]*User),
		Mux:         new(sync.RWMutex),
		CleanPeriod: opt.CleanPeriod,
		MsgTTL:      opt.MsgTTL,
	}

	go oracle.KeepClean()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if r.Method == "POST" {
			HandlePost(w, r, &oracle)
		} else {
			HandleConnection(w, r, &oracle)
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

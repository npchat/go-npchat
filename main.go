package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Info struct {
	Status     string    `json:"status"`
	StartTime  time.Time `json:"startTime"`
	DataLenMax int       `json:"dataLenMax"`
	MsgTTL     int       `json:"msgTtl"`
}

func main() {
	startTime := time.Now()

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
			return
		}
		if r.URL.Path == "/info" {
			handleInfo(w, &startTime, &opt)
			return
		}
		HandleConnection(w, r, &oracle, &opt)
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

func handleInfo(w http.ResponseWriter, startTime *time.Time, opt *Options) {
	w.Header().Add("Content-Type", "application/json")
	info, _ := json.Marshal(Info{
		Status:     "healthy",
		StartTime:  *startTime,
		DataLenMax: opt.DataLenMax,
		MsgTTL:     int(opt.MsgTTL.Seconds()),
	})
	w.Write(info)
}

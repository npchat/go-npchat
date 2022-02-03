package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Info struct {
	Status     string    `json:"status"`
	StartTime  time.Time `json:"startTime"`
	DataLenMax int       `json:"dataLenMax"`
	MsgTTL     int       `json:"msgTtl"`
	UserTTL    int       `json:"userTtl"`
}

func main() {
	startTime := time.Now()

	opt := LoadOptions()
	log.Printf("%+v'\n", opt)

	oracle := Oracle{
		Users:   make(map[string]*User),
		Mux:     new(sync.RWMutex),
		Options: &opt,
	}

	oracle.ReadState()

	go oracle.KeepClean()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if r.Method == "POST" {
			HandlePost(w, r, &oracle)
			return
		}
		if r.URL.Path == "/info" {
			handleGetInfo(w, &startTime, &opt)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/shareable") {
			HandleGetShareable(w, r, &oracle)
			return
		}
		HandleConnection(w, r, &oracle, &opt)
	})

	addr := fmt.Sprintf(":%v", opt.Port)
	log.Printf("listening on %v\n", addr)
	var err error
	if opt.CertFile != "" && opt.KeyFile != "" {
		log.Println("expecting HTTPS connections")
		err = http.ListenAndServeTLS(addr, opt.CertFile, opt.KeyFile, nil)
	} else {
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		log.Println("failed to start server", err)
	}
}

func GetIdFromPath(path string) string {
	// remove beginning "/"
	cleaned := strings.TrimLeft(path, "/")
	// return first segment of path
	return strings.Split(cleaned, "/")[0]
}

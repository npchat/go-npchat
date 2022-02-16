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

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal("failed to load config", err)
	}
	log.Printf("%+v'\n", cfg)

	kv := GobkvClient{
		mux:        new(sync.RWMutex),
		authSecret: cfg.Gobkv.AuthSecret,
	}
	go kv.keepClientUp(&cfg.Gobkv)

	oracle := Oracle{
		users:  make(map[string]*User),
		mux:    new(sync.RWMutex),
		config: &cfg,
		kv:     &kv,
	}

	go oracle.keepClean()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if r.Method == "POST" {
			handlePost(w, r, &oracle)
			return
		}
		if r.URL.Path == "/info" {
			handleGetInfo(w, &startTime, &cfg)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/shareable") {
			handleGetShareable(w, r, &oracle)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/turn") {
			if r.Method == "OPTIONS" {
				w.Header().Add("Access-Control-Allow-Headers", "Authorization")
				return
			}
			handleGetTurnInfo(w, r, &cfg.Turn)
			return
		}
		handleConnection(w, r, &oracle, &cfg)
	})

	addr := fmt.Sprintf(":%v", cfg.Port)
	log.Printf("listening on %v\n", addr)
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		log.Println("expecting HTTPS connections")
		err = http.ListenAndServeTLS(addr, cfg.CertFile, cfg.KeyFile, nil)
	} else {
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		log.Println("failed to start server", err)
	}
}

func getIdFromPath(path string) string {
	// remove beginning "/"
	cleaned := strings.TrimLeft(path, "/")
	// return first segment of path
	return strings.Split(cleaned, "/")[0]
}

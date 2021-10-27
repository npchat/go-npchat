package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Options struct {
	Port     int
	CertFile string
	KeyFile  string
}

func GetOptionsFromFlags() Options {
	o := Options{}
	flag.IntVar(&o.Port, "p", 8000, "port must be an int")
	flag.StringVar(&o.CertFile, "cert", "", "must be a relative file path")
	flag.StringVar(&o.KeyFile, "key", "", "must be a relative file path")
	flag.Parse()
	return o
}

func WriteToStore(firstLine string, record string) {
	if firstLine != "" {
		t := time.Now().In(time.UTC).GoString()
		err := ioutil.WriteFile(DBFILE, []byte(t+"\n"), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
	if record != "" {
		file, err := os.OpenFile(DBFILE, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		if _, err := file.WriteString(record + "\n"); err != nil {
			log.Fatal(err)
		}
	}
}

func CheckOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     CheckOrigin,
}

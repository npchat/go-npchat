package main

import (
	"flag"
)

type Options struct {
	Port     int
	CertFile string
	KeyFile  string
}

func GetOptionsFromFlags() Options {
	o := Options{}
	flag.IntVar(&o.Port, "p", 8000, "port must be an int")
	flag.StringVar(&o.CertFile, "cert", "cert.pem", "must be a relative file path")
	flag.StringVar(&o.KeyFile, "key", "key.pem", "must be a relative file path")
	flag.Parse()
	return o
}

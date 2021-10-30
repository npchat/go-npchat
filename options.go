package main

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Options struct {
	Port        int
	CertFile    string
	PrivKeyFile string
	MessageTTL  time.Duration
	CleanPeriod time.Duration
}

const PORT = 8000
const MSG_TTL = 60               // second
const CLEAN_PERIOD = MSG_TTL / 2 // second

func GetOptions() Options {
	// get ENV vars
	envCert := os.Getenv("NPCHAT_CERT")
	envPrivKey := os.Getenv("NPCHAT_PRIVKEY")

	envPort := os.Getenv("NPCHAT_PORT")
	defaultPort := PORT
	if envPort != "" {
		defaultPort, _ = strconv.Atoi(envPort)
	}

	envMsgTTL := os.Getenv("NPCHAT_MSG_TTL") // second
	defaultMsgTtl := MSG_TTL
	if envMsgTTL != "" {
		defaultMsgTtl, _ = strconv.Atoi(envMsgTTL)
	}
	envCleanPeriod := os.Getenv("NPCHAT_CLEAN_PERIOD") // second
	defaultCleanPeriod := CLEAN_PERIOD
	if envCleanPeriod != "" {
		defaultCleanPeriod, _ = strconv.Atoi(envCleanPeriod)
	}
	o := Options{}
	flag.StringVar(&o.CertFile, "cert", envCert, "must be a relative file path")
	flag.StringVar(&o.PrivKeyFile, "privkey", envPrivKey, "must be a relative file path")
	flag.IntVar(&o.Port, "p", defaultPort, "port must be an int")

	var argMsgTtl int
	flag.IntVar(&argMsgTtl, "msgttl", defaultMsgTtl, "port must be an int")

	var argCleanPeriod int
	flag.IntVar(&argCleanPeriod, "cleanperiod", defaultCleanPeriod, "port must be an int")
	flag.Parse()

	o.MessageTTL = time.Second * time.Duration(argMsgTtl)
	o.CleanPeriod = time.Second * time.Duration(argCleanPeriod)

	return o
}

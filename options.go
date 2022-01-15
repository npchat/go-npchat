package main

import (
	"flag"
	"os"
	"strconv"
	"time"
)

const PORT = 8000
const MSG_TTL = 60               // second
const CLEAN_PERIOD = MSG_TTL / 2 // second
const DATA_LEN_MAX = 2048        // 2MB

type Options struct {
	Port        int
	CertFile    string
	PrivKeyFile string
	MsgTTL      time.Duration
	CleanPeriod time.Duration
	DataLenMax  int
}

func LoadOptions() Options {
	envCert := os.Getenv("NPCHAT_CERT")
	envPrivKey := os.Getenv("NPCHAT_PRIVKEY")

	envPort := os.Getenv("NPCHAT_PORT")
	defaultPort := PORT
	if envPort != "" {
		defaultPort, _ = strconv.Atoi(envPort)
	}

	envDataLenMax := os.Getenv("NPCHAT_DATA_LEN_MAX")
	defaultDataLenMax := DATA_LEN_MAX
	if envDataLenMax != "" {
		defaultDataLenMax, _ = strconv.Atoi(envDataLenMax)
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
	flag.IntVar(&o.Port, "port", defaultPort, "port must be an int")
	flag.IntVar(&o.DataLenMax, "datalenmax", defaultDataLenMax, "datalenmax must be an int")

	var argMsgTtl int
	flag.IntVar(&argMsgTtl, "msgttl", defaultMsgTtl, "port must be an int")

	var argCleanPeriod int
	flag.IntVar(&argCleanPeriod, "cleanperiod", defaultCleanPeriod, "port must be an int")

	flag.Parse()

	o.MsgTTL = time.Second * time.Duration(argMsgTtl)
	o.CleanPeriod = time.Second * time.Duration(argCleanPeriod)

	return o
}

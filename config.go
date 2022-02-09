package main

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"
)

const envPrefix = "NPCHAT_"

const defaultPort = 8000
const defaultMsgTtl = time.Second * time.Duration(432000)   // 5 days
const defaultUserTtl = time.Second * time.Duration(7776000) // 90 days
const defaultCleanPeriod = time.Second * time.Duration(300) // 5 minutes
const defaultDataLenMax = 2048                              // 2MB

type Config struct {
	Port        int
	CertFile    string
	KeyFile     string
	MsgTTL      Duration
	UserTTL     Duration
	CleanPeriod Duration
	DataLenMax  int
	PersistFile string
}

// Correctly unmarshal duration in config file
// https://stackoverflow.com/a/48051946
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func LoadConfig() (Config, error) {
	envConfigFile := os.Getenv(envPrefix + "CONFIG")
	configFile := ""
	flag.StringVar(&configFile, "config", envConfigFile, "must be a file path")
	flag.Parse()
	cfg := Config{
		Port:        defaultPort,
		MsgTTL:      Duration{defaultMsgTtl},
		UserTTL:     Duration{defaultUserTtl},
		CleanPeriod: Duration{defaultCleanPeriod},
		DataLenMax:  defaultDataLenMax,
	}
	if configFile == "" {
		return cfg, nil
	} else {
		err := read(configFile, &cfg)
		return cfg, err
	}
}

func read(path string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

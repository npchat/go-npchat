package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func handleGetInfo(w http.ResponseWriter, startTime *time.Time, cfg *Config) {
	w.Header().Add("Content-Type", "application/json")
	info, _ := json.MarshalIndent(Info{
		Status:     "healthy",
		StartTime:  *startTime,
		DataLenMax: cfg.DataLenMax,
		MsgTTL:     int(cfg.MsgTTL.Seconds()),
		UserTTL:    int(cfg.UserTTL.Seconds()),
	}, "", "\t")
	w.Write(info)
}

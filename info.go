package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func handleGetInfo(w http.ResponseWriter, startTime *time.Time, opt *Options) {
	w.Header().Add("Content-Type", "application/json")
	info, _ := json.MarshalIndent(Info{
		Status:     "healthy",
		StartTime:  *startTime,
		DataLenMax: opt.DataLenMax,
		MsgTTL:     int(opt.MsgTTL.Seconds()),
		UserTTL:    int(opt.UserTTL.Seconds()),
	}, "", "\t")
	w.Write(info)
}

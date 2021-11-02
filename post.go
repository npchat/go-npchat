package main

import (
	"encoding/json"
	"io"
	"net/http"
)

func HandlePost(w http.ResponseWriter, r *http.Request, o *Oracle) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	id := GetIdFromPath(r.URL.Path)
	o.GetUser(id).Send(body, o.MsgTTL)
	resp := ServerResponse{Message: "sent"}
	rj, _ := json.Marshal(resp)
	w.Write(rj)
}

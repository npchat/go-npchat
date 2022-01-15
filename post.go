package main

import (
	"encoding/json"
	"io"
	"net/http"
)

func HandlePost(w http.ResponseWriter, r *http.Request, o *Oracle) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	id := GetIdFromPath(r.URL.Path)
	user, err := o.GetUser(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user.Send(body, o.MsgTTL)
	resp := ServerResponse{Message: "sent"}
	rj, _ := json.Marshal(resp)
	w.Write(rj)
}

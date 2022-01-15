package main

import (
	"encoding/json"
	"fmt"
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
	if !ValidateId(id) {
		http.Error(w, fmt.Sprintf("Invalid ID %v", id), http.StatusBadRequest)
		return
	}
	o.GetUser(id).Send(body, o.MsgTTL)
	resp := ServerResponse{Message: "sent"}
	rj, _ := json.Marshal(resp)
	w.Write(rj)
}

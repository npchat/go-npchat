package main

import (
	"io"
	"net/http"
)

func HandlePost(w http.ResponseWriter, r *http.Request, o *Oracle) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	id := GetIdFromPath(r.URL.Path)
	user, err := o.GetUser(id, true)
	if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	queryValues := r.URL.Query()
	doStore := queryValues.Get("store") != "false"

	user.Send(body, o.Config.MsgTTL.Duration, doStore)
}

package main

import (
	"io"
	"net/http"
)

func handlePost(w http.ResponseWriter, r *http.Request, oracle *Oracle) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	id := getIdFromPath(r.URL.Path)
	user, err := oracle.getUser(id, true)
	if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	queryValues := r.URL.Query()
	doStore := queryValues.Get("store") != "false"

	user.sendMessage(body, oracle, doStore)
}

package main

import (
	"net/http"
)

func handleGetShareable(w http.ResponseWriter, r *http.Request, o *Oracle) {
	id := getIdFromPath(r.URL.Path)

	w.Header().Add("Content-Type", "application/json")

	data, err := o.kv.get(id + "/shareable")
	if err != nil || len(data) == 0 {
		http.Error(w, "nothing found for id "+id, http.StatusNotFound)
	}

	w.Write(data)
}

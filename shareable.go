package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

func HandleGetShareable(w http.ResponseWriter, r *http.Request, o *Oracle) {
	idEnc := GetIdFromPath(r.URL.Path)

	id, err := base64.RawURLEncoding.DecodeString(idEnc)
	if err != nil || len(id) != 32 {
		http.Error(w, fmt.Sprintf("invalid id %v", idEnc), http.StatusBadRequest)
		return
	}

	user, err := o.GetUser(idEnc)
	if err != nil {
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(user.GetShareableData())
}

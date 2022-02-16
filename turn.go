package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TurnConfig struct {
	URL            string
	Secret         string
	CredentialsTtl Duration
}

type TurnInfo struct {
	URL        string `json:"urls"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
}

func makeCredential(username string, secret string) string {
	credH := hmac.New(sha1.New, []byte(secret))
	credH.Write([]byte(username))
	return base64.StdEncoding.EncodeToString(credH.Sum(nil))
}

func getTurnInfo(idEncoded string, cfg *TurnConfig) TurnInfo {
	expiry := time.Now().Add(cfg.CredentialsTtl.Duration)
	username := fmt.Sprintf("%d:%s", expiry.Unix(), idEncoded)
	return TurnInfo{
		URL:        cfg.URL,
		Username:   username,
		Credential: makeCredential(username, cfg.Secret),
	}
}

func handleGetTurnInfo(w http.ResponseWriter, r *http.Request, cfg *TurnConfig) {
	idEncoded := getIdFromPath(r.URL.Path)
	id, err := base64.RawURLEncoding.DecodeString(idEncoded)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	authMsg, err := getAuthMsgFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !verifyAuthMessage(&authMsg, id) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	turnInfo := getTurnInfo(idEncoded, cfg)
	resp, _ := json.Marshal(turnInfo)
	w.Write(resp)
	w.Header().Add("Content-Type", "application/json")
}

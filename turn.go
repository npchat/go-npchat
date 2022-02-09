package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/shamaton/msgpack/v2"
)

type TurnConfig struct {
	Servers []TurnServerCfg
}

type TurnServerCfg struct {
	URL            string
	Secret         string
	CredentialsTtl Duration
}

type TurnServerResponse struct {
	URL      string `json:"urls"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func getAuthMsgFromRequest(w http.ResponseWriter, r *http.Request) (AuthMessage, error) {
	authMsg := AuthMessage{}
	authHeader := r.Header.Get("Authorization")
	authHeaderDecoded, err := base64.RawURLEncoding.DecodeString(authHeader)
	if err != nil {
		http.Error(w, "invalid authorization header", http.StatusBadRequest)
		return authMsg, err
	}

	err = msgpack.Unmarshal(authHeaderDecoded, &authMsg)
	if err != nil {
		http.Error(w, "invalid authorization header", http.StatusBadRequest)
		return authMsg, err
	}

	return authMsg, nil
}

func HandleGetTurnServers(w http.ResponseWriter, r *http.Request, cfg *Config) {
	idEncoded := GetIdFromPath(r.URL.Path)
	id, err := base64.RawStdEncoding.DecodeString(idEncoded)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// authenticate request
	authMsg, _ := getAuthMsgFromRequest(w, r)
	if !VerifyAuthMessage(&authMsg, id) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// build credentials response, an array of credentials
	// one for each TurnServer in TurnConfig
	turnServers := make([]TurnServerResponse, len(cfg.Turn.Servers))

	for i, serverCfg := range cfg.Turn.Servers {
		ttl := time.Now().Add(serverCfg.CredentialsTtl.Duration)
		timestamp := strconv.FormatInt(ttl.UnixMilli(), 10)
		username := timestamp + ":" + idEncoded

		// MAYBE hash secret using MD5... read Coturn docs
		h := hmac.New(sha256.New, []byte(serverCfg.Secret))
		h.Write([]byte(username))
		password := base64.StdEncoding.EncodeToString(h.Sum(nil))

		turnServers[i] = TurnServerResponse{
			URL:      serverCfg.URL,
			Username: username,
			Password: password,
		}
	}

	resp, _ := json.MarshalIndent(turnServers, "", "  ")
	w.Write(resp)
	w.Header().Add("Content-Type", "application/json")
}

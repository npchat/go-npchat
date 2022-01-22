package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"log"
	"math/big"
	"strconv"
	"time"
)

func VerifyAuthMessage(msg *AuthMessage, id []byte) bool {
	// verify Time is within 1 second
	threshold := time.Duration(1) * time.Second
	timestamp, err := strconv.ParseInt(string(msg.Time), 10, 64)
	if err != nil {
		log.Println("failed to convert time to int")
	}
	timeDec := time.UnixMilli(timestamp)

	if timeDec.Before(time.Now()) {
		// check if not too old
		if timeDec.Add(threshold).Before(time.Now()) {
			return false
		}
	} else {
		// check if not too far in future
		if time.Now().Add(threshold).Before(timeDec) {
			return false
		}
	}

	// check id equals SHA-256 of public Key
	h := sha256.New()
	h.Write(msg.PublicKey)
	pubHash := h.Sum(nil)
	if !bytes.Equal(id, pubHash) {
		log.Println("public key does not match id", id, pubHash)
		return false
	}

	// unmarshal client public key
	pubKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(msg.PublicKey[1:33]),
		Y:     new(big.Int).SetBytes(msg.PublicKey[33:])}

	// hash Time with SHA-256
	tH := sha256.New()
	tH.Write(msg.Time)
	tHash := tH.Sum(nil)

	// verify client signature
	cL := len(msg.Sig) / 2
	cSigR := new(big.Int).SetBytes(msg.Sig[:cL])
	cSigS := new(big.Int).SetBytes(msg.Sig[cL:])

	return ecdsa.Verify(&pubKey, tHash, cSigR, cSigS)
}

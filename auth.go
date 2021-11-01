package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"math/big"
)

const CHALLENGE_LEN = 32

type Challenge struct {
	Txt string `json:"txt"`
	Sig string `json:"sig"`
}

type ChallengeWrapper struct {
	Challenge Challenge `json:"challenge"`
}

func genRandomBytes(size int) (blk []byte, err error) {
	blk = make([]byte, size)
	_, err = rand.Read(blk)
	if err != nil {
		log.Println(err)
	}
	return
}

func GetChallenge(priv *ecdsa.PrivateKey) (ChallengeWrapper, error) {
	randBytes, err := genRandomBytes(CHALLENGE_LEN)
	if err != nil {
		return ChallengeWrapper{}, err
	}
	h := sha256.New()
	h.Write(randBytes)
	randBytesHash := h.Sum(nil)
	prng := rand.Reader
	r, s, err := ecdsa.Sign(prng, priv, randBytesHash)
	if err != nil {
		return ChallengeWrapper{}, err
	}
	sigBytes := []byte{}
	sigBytes = append(sigBytes, r.Bytes()...)
	sigBytes = append(sigBytes, s.Bytes()...)
	sigStr := base64.RawURLEncoding.EncodeToString(sigBytes)
	txt := base64.RawURLEncoding.EncodeToString(randBytesHash)
	chall := Challenge{txt, sigStr}
	return ChallengeWrapper{Challenge: chall}, nil
}

func VerifySolution(msg *ClientMessage, id []byte, sPub *ecdsa.PublicKey) bool {
	// decode client public key
	cPubBytes, err := base64.RawURLEncoding.DecodeString(msg.PublicKey)
	if err != nil {
		log.Println(err)
		return false
	}

	// check id equals SHA-256 of public Key
	h := sha256.New()
	h.Write(cPubBytes)
	cPubHash := h.Sum(nil)
	if !bytes.Equal(id, cPubHash) {
		log.Println("public key does not match id", id, cPubHash)
		return false
	}

	// decode challenge
	txt, err := base64.RawURLEncoding.DecodeString(msg.Challenge.Txt)
	if err != nil {
		log.Println(err)
		return false
	}

	// verify server signature
	sSig, err := base64.RawURLEncoding.DecodeString(msg.Challenge.Sig)
	if err != nil {
		log.Println(err)
		return false
	}
	sL := len(sSig) / 2
	sSigR := new(big.Int).SetBytes(sSig[:sL])
	sSigS := new(big.Int).SetBytes(sSig[sL:])
	sValid := ecdsa.Verify(sPub, txt, sSigR, sSigS)
	if !sValid {
		log.Println("server signature invalid")
		return false
	}

	// unmarshal client public key
	cPub := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(cPubBytes[1:33]),
		Y:     new(big.Int).SetBytes(cPubBytes[33:])}

	// verify client signature
	cSig, err := base64.RawURLEncoding.DecodeString(msg.Solution)
	if err != nil {
		log.Println(err)
		return false
	}
	cL := len(cSig) / 2
	cSigR := new(big.Int).SetBytes(cSig[:cL])
	cSigS := new(big.Int).SetBytes(cSig[cL:])

	// hash challenge with SHA-256
	txtH := sha256.New()
	txtH.Write(txt)
	txtHash := txtH.Sum(nil)

	return ecdsa.Verify(&cPub, txtHash, cSigR, cSigS)
}

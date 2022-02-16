package main

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"hash/fnv"
	"log"
	"net/rpc"
	"sync"
	"time"

	"github.com/dr-useless/gobkv/common"
)

type GobkvConfig struct {
	Address    string
	AuthSecret string
	CertFile   string
	KeyFile    string
}

// Keeps a connection to gobkv up
// Lock is for getting client ready
// Rlock are for normal operations
type GobkvClient struct {
	client     *rpc.Client
	mux        *sync.RWMutex
	authSecret string
}

func (kv *GobkvClient) get(key string) ([]byte, error) {
	kv.mux.RLock()
	defer kv.mux.RUnlock()
	rpcArgs := common.Args{
		AuthSecret: kv.authSecret,
		Key:        key,
	}
	var reply common.ValueReply
	err := kv.client.Call("Store.Get", rpcArgs, &reply)
	return reply.Value, err
}

func (kv *GobkvClient) set(key string, value []byte) error {
	kv.mux.RLock()
	defer kv.mux.RUnlock()
	rpcArgs := common.Args{
		AuthSecret: kv.authSecret,
		Key:        key,
		Value:      value,
	}
	var reply common.StatusReply
	return kv.client.Call("Store.Set", rpcArgs, &reply)
}

// Asynchronously set value with automatic key.
// Key is given prefix + base64 hash of value.
// Returns key & done channel
func (kv *GobkvClient) setAuto(prefix string, value []byte) (string, chan *rpc.Call) {
	h := fnv.New64a()
	h.Write(value)
	key := prefix + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	kv.mux.RLock()
	defer kv.mux.RUnlock()
	rpcArgs := common.Args{
		AuthSecret: kv.authSecret,
		Key:        key,
		Value:      value,
	}
	var reply common.StatusReply
	var done chan *rpc.Call
	kv.client.Go("Store.Set", rpcArgs, &reply, done)
	return key, done
}

func (kv *GobkvClient) list(prefix string) ([]string, error) {
	kv.mux.RLock()
	defer kv.mux.RUnlock()
	rpcArgs := common.Args{
		AuthSecret: kv.authSecret,
		Key:        prefix,
	}
	var reply common.KeysReply
	err := kv.client.Call("Store.List", rpcArgs, &reply)
	return reply.Keys, err
}

func (kv *GobkvClient) del(key string) error {
	kv.mux.RLock()
	defer kv.mux.RUnlock()
	rpcArgs := common.Args{
		AuthSecret: kv.authSecret,
		Key:        key,
	}
	var reply common.StatusReply
	return kv.client.Call("Store.Del", rpcArgs, &reply)
}

func (kv *GobkvClient) getClient(cfg *GobkvConfig) error {
	if cfg.CertFile == "" {
		// return client on open tcp connection
		client, err := rpc.Dial("tcp", cfg.Address)
		if err != nil {
			return err
		}
		kv.client = client
	} else {
		// load cert & key
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			log.Fatalf("failed to load key pair: %s", err)
		}
		config := tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		// return client on tls connection
		conn, err := tls.Dial("tcp", cfg.Address, &config)
		if err != nil {
			return err
		}
		kv.client = rpc.NewClient(conn)
	}
	return nil
}

func (kv *GobkvClient) ping() error {
	rpcArgs := common.Args{} // try giving empty interface
	var reply common.StatusReply
	err := kv.client.Call("Store.Ping", rpcArgs, &reply)
	if err != nil {
		return err
	}
	if reply.Status != common.StatusOk {
		return errors.New("ping reply was not OK")
	}
	return nil
}

func (kv *GobkvClient) keepClientUp(cfg *GobkvConfig) {
	err := kv.getClient(cfg)
	if err != nil {
		log.Fatal("failed to get gobkv client conn:", err)
	}
	log.Println("connected to gobkv at", cfg.Address)
	printedConnError := false
	for {
		if err := kv.ping(); err != nil {
			if !printedConnError {
				log.Println("dropped connection to gobkv, will try to reconnect...")
				printedConnError = true
			}
			kv.mux.Lock()
			kv.client.Close()
			err = kv.getClient(cfg)
			if err == nil {
				log.Println("reconnected to gobkv")
				printedConnError = false
			}
			kv.mux.Unlock()
		}
		time.Sleep(time.Duration(15) * time.Second)
	}
}

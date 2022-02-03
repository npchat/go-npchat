# go-npchat
A lightweight npchat server written in Go.

### Simplicity
The goal of this project is to implement the simplest possible solution for federated & secure communication.

The result is a binary or tiny container (< 10MB) that can be deployed anywhere, at any scale.

## Usage
### With Docker
```zsh
% make build
```
Then
```zsh
% docker run go-npchat:latest
```
### With Go
Build a binary
```zsh
% go build
```
Or from source
```zsh
% go run .
```

## Configuration
Configure the chat server using either environment variables or arguments.
- Port `default 8000`
- Cert `default ""`
- PrivKey `default ""`
- MessageTTL `default 43200 seconds`
- UserTTL `default 7776000 seconds`
- CleanPeriod `default 300 seconds`
- DataLenMax `default 2048`
- PersistFile `default ./persist.json`

If no SSL Cert & Key is provided, the HTTP server will start without TLS.

### Environment variables
```zsh
export NPCHAT_PORT=8000
export NPCHAT_CERT=""
export NPCHAT_KEY=""
export NPCHAT_MSG_TTL=43200
export NPCHAT_CLEAN_PERIOD=43200
```
### Arguments
```zsh
% ./go-npchat --port=443 \
  --cert="cert.pem" --key="key.pem" \
  --msgttl=43200 --userttl=43200
  --cleanperiod=300
```

## Transport
### Incoming
Incoming messages must be sent with a simple `POST` request. This allows clients to send messages to many recipients concurrently, without the overhead & complexity of managing multiple long-lived connections.

A message post request should look something like:
```zsh
% curl 'https://go.npchat.org:8000/WoQSzF_Hp_sC_nv03Wv6HofbLM1iY6taN9lZ2NZyrZg' -X POST \
--data-raw '{"t":1635605638136, \
"iv":"boTInSlEnz2CP649PevG_4EyPnikvJ0ODScGZtg1TCA", \
"m":"knSOiIT4t06W7QVpl3qQ4UZyL6tN", \
"f":"4es6LG3vF6hDs9-AIvbc-xII4d0lWmmSseMDfF2bNlY", \
"h":"ioKCJdSmvglamXV-6OdU4xoXK3_V8QwRIuE1TLXYHR0", \
"p":"iSM1biGkutj5Y4AcWViOzpU1XJl9y5wJjlNettmZXdY", \
"s":"Rg0bC5zPMcOW1UbZMdcF7NBKZMLOVlPqG_zgRRG_ztkdK7nswQgmWaMEpQQw6HU5KMQICX3GUI6mE0uwBkj8lg"}'
```
#### Public key hash
The key peice of information in this request is the URL pathname `WoQSzF_Hp_sC_nv03Wv6HofbLM1iY6taN9lZ2NZyrZg`.
This is the hash of the recipient's ECDSA P-256 public key.

#### Message body
The only requirement for the message body, `--data-raw`, is that it must be a string. It is not parsed at all by the chat server, so clients are free to impelement any kind of messaging features.


### Outgoing
All messages to a reciepient (from any sender) will hit the same origin domain set by the recipient. This could be a load-balanced cluster of nodes, or a single instance.

A Client connects to the chat server & recieves messages as follows:
1. Client requests a WebSocket upgrade.
2. Client requests challenge from Server
3. Client signs challenge, and returns it to Server along with their public key
4. Server verifies that:
  - Hash of public key = publicKeyHash in URL pathname
  - Client signature is valid
  - Server signature is valid
5. Server sends a message that authentication is done `{"message": "handshake done"}`
6. Server sends all messages that have not yet been delivered (and not expired & cleaned up)
7. Server forwards any message recieved immediately, without storing

If a Client connection ends, their session is unregistered and messages will be stored until either they are delivered, or they expire and are deleted.
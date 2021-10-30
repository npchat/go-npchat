# go-npchat
A lightweight npchat server written in Go.

Containerized using first `golang:alpine`, then `FROM scratch` to reduce image size to around 10MB.

### Simplicity
The goal of this project is to implement the simplest possible solution for federated & secure communication.

The result is a binary or container that can be deployed anywhere, at any scale.

## Usage
### With Docker
```zsh
% make build
```
Then
```zsh
% docker run -p 8000 go-npchat:latest
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
- MessageTTL `default 60 seconds`
- CleanPeriod `default 30 seconds`

If no Cert & PrivKey is provided, the HTTP server will start without TLS.

### Environment variables
```zsh
export NPCHAT_PORT=8000
export NPCHAT_CERT=""
export NPCHAT_PRIVKEY=""
export NPCHAT_MSG_TTL=60
export NPCHAT_CLEAN_PERIOD=30
```
### Arguments
```zsh
% ./go-npchat --port=443 \
  --cert="cert.pem" --privkey="privkey.pem" \
  --msgttl=60 --cleanperiod=30
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
The key peice of information in this request is `WoQSzF_Hp_sC_nv03Wv6HofbLM1iY6taN9lZ2NZyrZg`.
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

## Reliability
By removing the fundamental requirement that a chat server must store messages until collected (indefinitely), we can build a very simple solution that must not ensure that messages are guaranteed to persist. This greatly lowers the resource requirements of the chat server. 

So, if the chat server cannot guarantee that it will deliver messages to an offline recipient (due to storage expiry), how can we guarantee that all messages will be delivered at some point?

The solution is actually not part of the chat server, it's part of the client. If each message contains the hash of the preious message (recieved or sent), both end parties can detect a missing message. If a missing message is detect 

## Security
### Authentication
With this mechanism, security depends on privacy of the Server's auth private key & privacy of the Client's auth private key. A client is obviously responsible for secure storage of it's keys.

Every challenge contains the signature for the random bytes. An attacker can request as many challenges as they want...

So how do we prevent the private key being deduced by requesting enough signed challenges? This is a fundamental vulnerability if not handled carefully.

Adding a timestamp or TTL to the signed challenge would ensure that challenges do not remain valid indefinitely, but it does not prevent forged authorization when the private key is deduced by collecting enough challenges.

My answer to this question is extremly simple. After `x` challenges are served, generate a fresh key pair. If `x` is low enough that an attacker cannot get enough challenges to deduce the private key, his efforts are futile. This solution does not require any refresh period to be specified, and does not alter in behaviour with increased traffic. The default configuration is `x=5`, because generating P-256 key pairs is computationally inexpensive. This will probably change after some benchmarking & research.

If the assumptions based on a just little knowledge of cryptography & a bunch of research is correct, then the following claims would be true: 
- Messages cannot be forged or modified
- Messages can only be collected by the holder of the private key corresponding to a given public key

### Privacy
Out of the box, this chat server should provide secure & authenticated transport for messages.

In reality, that has nothing to do with the content that is being transmitted. As such, complete decoupling here is the most secure & separated approach. Hence, encryption of messages is handled completely & only by the client.

This ensures that clients are flexible in what they send, and also flexible in how they send it. Currently the only constraint is that messages are `text/plain` content, normally stringified JSON. A goal is to remove this constraint to allow clients to do things like negotiate WebRTC connections more easily.

#### How the webclient does it
The npchat webclient implements ECDH P-256 to derive a shared secret from the ECDH public keys, and AES-GCM to encrypt messages with the shared secret. A more complete key derivation step is necessary here, HKDF would be good.

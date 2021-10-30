# go-npchat
A lightweight npchat server implementation written in Go.

Containerized using first `golang:alpine`, then `FROM scratch` to reduce image size to around 10MB.

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

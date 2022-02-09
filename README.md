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
% docker run go-npchat
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

## To do
- Return ephemeral TURN credentials upon request
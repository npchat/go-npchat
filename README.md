# go-npchat
A lightweight npchat server implementation written in Go.

It's already containerized using a 2-stage build process. First `golang:alpine`, then `FROM scratch` to minimise image size.

## Installation requirements
- Docker

## Usage
```zsh
% make build
```
Build should take around 10s or less, and come in at less than 10MB.
```zsh
```zsh
% docker run -p 8000 go-npchat:latest
```

## Protocol
For details about the protocol, please see the original repo [npchat](https://github.com/dr-useless/npchat).
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
Either give go-npchat a `.json` configuration file, or let it run with defaults. If any fields are ommited from the config, the default is used.

### Env
```zsh
export NPCHAT_CONFIG="config.json"
```
### Arg
```zsh
% ./go-npchat --config "config.json"
```

### Fields
See below the fields & their default values.
```json
{
  "Port": 8000,
  "CertFile": "",
  "KeyFile": "",
  "MsgTtl": "120h",
  "UserTtl": "2160h",
  "CleanPeriod": "5m",
  "DataLenMax": 2048,
  "PersistFile": ""
}
```

## To do
- Return ephemeral TURN credentials upon request
## Based on
## https://github.com/chemidy/smallest-secured-golang-docker-image

VERSION=`git rev-parse HEAD`
BUILD=`date +%FT%T%z`
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"
DOCKER_IMAGE=go-npchat

## - Show help message
.PHONY: help
help:
	@printf "\033[32m\xE2\x9c\x93 usage: make [target]\n\n\033[0m"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

## - Docker pull latest images
.PHONY: docker-pull
docker-pull:
	@printf "\033[32m\xE2\x9c\x93 docker pull latest images\n\033[0m"
	@docker pull golang:alpine

## - Build go-npchat
.PHONY: build
build:docker-pull
	@printf "\033[32m\xE2\x9c\x93 Build go-npchat\n\033[0m"
	$(eval BUILDER_IMAGE=$(shell docker inspect --format='{{index .RepoDigests 0}}' golang:alpine))
	@export DOCKER_CONTENT_TRUST=1
	@docker build -f docker/scratch.Dockerfile --build-arg "BUILDER_IMAGE=$(BUILDER_IMAGE)" -t go-npchat .

## - List go-npchat docker images
.PHONY: ls
ls:
	@printf "\033[32m\xE2\x9c\x93 Look at the size dude !\n\033[0m"
	@docker image ls go-npchat

## - Run go-npchat
.PHONY: run
run:
	@printf "\033[32m\xE2\x9c\x93 Run go-npchat\n\033[0m"
	@docker run -p 8000 go-npchat --cert cert.pem --key key.pem

## - Scan for known vulnerabilities
.PHONY: scan
scan:
	@printf "\033[32m\xE2\x9c\x93 Scan for known vulnerabilities go-npchat\n\033[0m"
	@docker scan -f Dockerfile go-npchat

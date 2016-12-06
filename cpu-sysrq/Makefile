GO := go
pkgs  = $(shell $(GO) list ./...)
tag = $(shell git rev-parse --short HEAD)

all: build docker

build:
	@CGO_ENABLED=0 $(GO) build -o cpu-sysrq # -tags netgo -a -v

docker:
	@docker build -t vish/cpu-sysrq:$(tag) .

push:
	@echo ">>pushing vish/cpu-sysrq:$(tag)"
	@docker push vish/cpu-sysrq:$(tag)

push-latest:
	@docker tag vish/cpu-sysrq:$(tag) vish/cpu-sysrq:latest
	@docker push vish/cpu-sysrq:latest

.PHONY: all build docker push

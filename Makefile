# variable definitions
NAME := platform
DESC := core of the Reconfigure.io platform
PREFIX ?= usr/local
VERSION := $(shell git describe --tags --always --dirty)
GOVERSION := $(shell go version)
BUILDTIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILDDATE := $(shell date -u +"%B %d, %Y")
BUILDER := $(shell echo "`git config user.name` <`git config user.email`>")
PKG_RELEASE ?= 1
PROJECT_URL := "https://github.com/ReconfigueIO/$(NAME)"
LDFLAGS := -X 'main.version=$(VERSION)' \
           -X 'main.buildTime=$(BUILDTIME)' \
           -X 'main.builder=$(BUILDER)' \
           -X 'main.goversion=$(GOVERSION)'

.PHONY: test install clean all

CMD_SOURCES := $(shell find cmd -name main.go)
TARGETS := $(patsubst cmd/%/main.go,dist-image/dist/%,$(CMD_SOURCES))

all: ${TARGETS} dist-image/dist/main

test: fmt
	go test -v $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

install:
	glide install

dist-image/dist:
	mkdir -p dist

dist-image/dist/%: cmd/%/main.go | dist-image/dist
	go build -ldflags "$(LDFLAGS)" -o $@ $<

dist-image/dist/main: main.go | dist-image/dist
	go build -ldflags "$(LDFLAGS)" -o $@ $<

clean:
	rm -rf dist-image/dist

image: all
	docker build -t "reco-api:latest" dist-image

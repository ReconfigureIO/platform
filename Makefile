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

.PHONY: test install clean all generate

CMD_SOURCES := $(shell find cmd -name main.go)
TARGETS := $(patsubst cmd/%/main.go,dist-image/dist/%,$(CMD_SOURCES))

TEMPLATE_SOURCES := $(shell find templates -name *.tmpl)
TEMPLATE_TARGETS := $(patsubst templates/%,dist-image/dist/templates/%,$(TEMPLATE_SOURCES))


all: ${TARGETS} ${TEMPLATE_TARGETS} dist-image/dist/main

generate:
	go generate -v $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

test: generate
	go test -v $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

install:
	glide install && go test -i $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

dist-image/dist:
	@mkdir -p $@

dist-image/dist/%: cmd/%/main.go | dist-image/dist
	go build -ldflags "$(LDFLAGS)" -o $@ $<

dist-image/dist/main: main.go | dist-image/dist
	go build -ldflags "$(LDFLAGS)" -o $@ $<

dist-image/dist/templates: dist-image/dist
	@mkdir -p $@

dist-image/dist/templates/%: templates/% | dist-image/dist/templates
	@cp $< $@

clean:
	rm -rf dist-image/dist

image: all
	docker build -t "reco-api:latest" dist-image
	docker build -t "reco-api:latest-worker" dist-worker

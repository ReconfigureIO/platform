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
PROJECT_URL := "https://github.com/ReconfigureIO/$(NAME)"
LDFLAGS := -X 'main.version=$(VERSION)' \
           -X 'main.buildTime=$(BUILDTIME)' \
           -X 'main.builder=$(BUILDER)' \
           -X 'main.goversion=$(GOVERSION)'

.PHONY: test install clean all generate deploy-production deploy-staging push-image image vet integration-tests

CMD_SOURCES := $(shell find cmd -name main.go)
TARGETS := $(patsubst cmd/%/main.go,dist-image/dist/%,$(CMD_SOURCES))

TEMPLATE_SOURCES := $(shell find templates -name *.tmpl)
TEMPLATE_TARGETS := $(patsubst templates/%,dist-image/dist/templates/%,$(TEMPLATE_SOURCES))

DOCKER_TAG := ${VERSION}
DOCKER_IMAGE := 398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api

all: ${TARGETS} ${TEMPLATE_TARGETS} dist-image/dist/main

vet:
	go list ./... | grep -v /vendor/ | xargs -L1 go vet -v

lint:
	go list ./... | grep -v /vendor/ | xargs -L1 golint

errcheck:
	go list ./... | grep -v /vendor/ | xargs -L1 errcheck

dependencies:
	glide install

$(GOPATH)/bin/mockgen: dependencies $(shell find vendor/github.com/golang/mock -name \*.go)
	cd vendor/github.com/golang/mock/mockgen && \
	go get `glide list 2> /dev/null | grep -A100 MISSING | grep -v MISSING | awk '{$$1=$$1};1'` && \
	go build -o $(GOPATH)/bin/mockgen

generate: $(GOPATH)/bin/mockgen
	go generate -v $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

test:
	go test -v $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

integration-tests:
	go test -tags=integration -v $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

install: generate
	go test -i $$(go list ./... | grep -v /vendor/ | grep -v /cmd/)

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
	find . -name '*_mock.go' -delete

image: all
	docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} dist-image

push-image:
	$$(aws ecr get-login --region us-east-1)
	docker push ${DOCKER_IMAGE}:${DOCKER_TAG}

migrate-production:
	kubectl patch -o yaml -f k8s/migrate_production.yml --local=true --type=json -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"${DOCKER_IMAGE}:${DOCKER_TAG}"}]' | kubectl create -f -
	./ci/wait_for.sh job migrate-production
	kubectl logs job/migrate-production
	kubectl delete job migrate-production

deploy-production:
	kubectl rollout pause deployment production-platform-web
	kubectl rollout pause deployment production-platform-cron

	kubectl apply -f k8s/production/

	kubectl set image -f k8s/production/api.yml api=${DOCKER_IMAGE}:${DOCKER_TAG}
	kubectl set image -f k8s/production/cron.yml cron=${DOCKER_IMAGE}:${DOCKER_TAG}

	kubectl rollout resume deployment production-platform-web
	kubectl rollout resume deployment production-platform-cron

	kubectl rollout status deployment production-platform-web
	kubectl rollout status deployment production-platform-cron

migrate-staging:
	kubectl patch -o yaml -f k8s/migrate_staging.yml --local=true --type=json -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"${DOCKER_IMAGE}:${DOCKER_TAG}"}]' | kubectl create -f -
	./ci/wait_for.sh job migrate-staging
	kubectl logs job/migrate-staging
	kubectl delete job migrate-staging

deploy-staging:
	kubectl rollout pause deployment staging-platform-web
	kubectl rollout pause deployment staging-platform-cron

	kubectl apply -f k8s/staging/

	kubectl set image -f k8s/staging/api.yml api=${DOCKER_IMAGE}:${DOCKER_TAG}
	kubectl set image -f k8s/staging/cron.yml cron=${DOCKER_IMAGE}:${DOCKER_TAG}

	kubectl rollout resume deployment staging-platform-web
	kubectl rollout resume deployment staging-platform-cron

	kubectl rollout status deployment staging-platform-web
	kubectl rollout status deployment staging-platform-cron

compose-test:
	(docker-compose rm -f -s db || 0) && docker-compose run --rm test bash -c "go test -v ${ARGS}"

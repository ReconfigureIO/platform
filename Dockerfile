# syntax = docker/dockerfile:experimental
FROM golang:1.11.2-alpine AS builder

ARG RECO_PLATFORM_VERSION=unknown
ARG RECO_PLATFORM_BUILDER=unknown

ENV GO111MODULE=on

RUN --mount=type=cache,target=/var/cache/apk,id=apk \
    apk add git alpine-sdk

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    go mod download

COPY . ./

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
\
    go install ./...

FROM scratch AS runtime

ENV PATH=/go/bin
COPY --from=builder /go/bin /go/bin
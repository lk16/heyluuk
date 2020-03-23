FROM golang:1.14-alpine

RUN apk update && apk upgrade && apk add --no-cache bash

LABEL maintainer="Luuk Verweij <luuk_verweij@msn.com>"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

ADD ./cmd ./cmd
ADD ./internal ./internal

ENV CGO_ENABLED 0
RUN go install ./cmd/heyluuk

ADD ./web ./web

CMD heyluuk
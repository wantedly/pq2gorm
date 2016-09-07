FROM golang:1.7.0-alpine

ENV GOPATH /go
WORKDIR /go/src/github.com/wantedly/pq2gorm

RUN apk add --no-cache --update bash curl git make

COPY Makefile /go/src/github.com/wantedly/pq2gorm/
COPY glide.yaml /go/src/github.com/wantedly/pq2gorm/
COPY glide.lock /go/src/github.com/wantedly/pq2gorm/

RUN make deps

COPY . /go/src/github.com/wantedly/pq2gorm

RUN make

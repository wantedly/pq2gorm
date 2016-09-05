NAME := pq2gorm
SRC := $(shell find . -type f -name "*.go")
LDFLAGS := -ldflags="-s -w"

GLIDE := $(shell command -v glide 2> /dev/null)

.DEFAULT_GOAL := bin/$(NAME)

bin/$(NAME): deps $(SRC)
	go build $(LDFLAGS) -o bin/$(NAME)

.PHONY: clean
clean:
	rm -rf bin/*
	rm -rf vendor/*

.PHONY: deps
deps: glide
	glide install

.PHONY: glide
glide:
ifndef GLIDE
	curl https://glide.sh/get | sh
endif

.PHONY: install
install:
	go install $(LDFLAGS)

.PHONY: update-deps
update-deps: glide
	glide update

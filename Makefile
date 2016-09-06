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

.PHONY: test
test:
	docker-compose stop
	docker-compose rm -f
	docker-compose up -d db
	sleep 5
	docker-compose exec db psql -U postgres -d test -f /testdata/db.dump
	docker-compose run --rm pq2gorm 'postgres://postgres:password@db:5432/test?sslmode=disable' -d /out

.PHONY: update-deps
update-deps: glide
	glide update

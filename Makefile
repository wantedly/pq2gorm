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

.PHONY: generate-test
generate-test:
	@docker-compose stop > /dev/null
	@docker-compose rm -f > /dev/null
	docker-compose up -d db
	script/ping_db.sh
	docker-compose exec db psql -U postgres -d test -f /testdata/testdata.sql
	docker-compose build pq2gorm
	docker-compose run --rm pq2gorm script/test.sh
	@docker-compose stop > /dev/null
	@docker-compose rm -f > /dev/null

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
	go test -v

.PHONY: update-deps
update-deps: glide
	glide update

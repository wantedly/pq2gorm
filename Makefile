NAME := pq2gorm
LDFLAGS := -ldflags="-s -w"

.DEFAULT_GOAL := bin/$(NAME)

bin/$(NAME): deps
	go generate
	go build $(LDFLAGS) -o bin/$(NAME)

.PHONY: clean
clean:
	rm -rf bin/*
	rm -rf vendor/*

.PHONY: deps
deps: glide
	go get github.com/jteeuwen/go-bindata/...
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
ifeq ($(shell command -v glide 2> /dev/null),)
	curl https://glide.sh/get | sh
endif

.PHONY: install
install:
	go generate
	go install $(LDFLAGS)

.PHONY: test
test:
	go generate
	go test -v

.PHONY: update-deps
update-deps: glide
	glide update

export SHELL:=/bin/bash

.PHONY: build
build:
	cd test-assets/lambda && GOOS=linux go build -o main .

.PHONY: test
test: build
	go test . ./... -coverprofile=coverage.out -coverpkg=./...
	go tool cover -html=coverage.out -o ./coverage.html

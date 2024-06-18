export SHELL:=/bin/bash

.PHONY: build
build:
	cd test-assets/lambda && rm main && GOOS=linux CGO_ENABLED=0 go build -o main main.go

.PHONY: test
test: build
	go test . ./... -coverprofile=coverage.out -coverpkg=./...
	go tool cover -html=coverage.out -o ./coverage.html

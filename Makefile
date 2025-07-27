# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=kalo
BINARY_UNIX=$(BINARY_NAME)_unix

.PHONY: all build clean test deps run

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./src

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) download
	$(GOMOD) tidy

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./src
	./$(BINARY_NAME)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./src

install: build
	cp $(BINARY_NAME) /usr/local/bin/
# Current version of the project.
VERSION ?= 0.3.0+git

# This repo's root import path (under GOPATH).
ROOT := github.com/tensorflow/k8s

# Project main package location (can be multiple ones).
CMD_DIR := ./cmd/tf_operator

# Project output directory.
OUTPUT_DIR := ./bin

# Git commit sha.
GitSHA := $(shell git rev-parse --short HEAD)

# Golang standard bin directory.
BIN_DIR := $(GOPATH)/bin

build:
	go build -i -v -o $(OUTPUT_DIR)/tf_operator                             \
	  -ldflags "-s -w -X $(ROOT)/version.Version=$(VERSION)      \
	            -X $(ROOT)/version.GitSHA=$(GitSHA)"            \
	  $(CMD_DIR)                                                     \

clean:
	-rm -vrf ${OUTPUT_DIR}

.PHONY: clean build
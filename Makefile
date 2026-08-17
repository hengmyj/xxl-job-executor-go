OUTPUT ?= bin/xxl-job-executor

.PHONY: build run rebuild test clean

build:
	./build.sh

run:
	./run.sh

rebuild:
	./run.sh --rebuild

test:
	go test ./...

clean:
	rm -rf bin

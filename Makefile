.PHONY: all-tests test gosec lint

all-tests: test gosec lint

test:
	go test ./... -covermode=atomic -race

gosec:
	gosec -stdout -verbose=text ./...

lint:
	golangci-lint  run
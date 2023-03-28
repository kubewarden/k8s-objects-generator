.PHONY: build
build:
	go build

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -v -coverprofile cover.out ./...

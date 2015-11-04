NAME=marathon-service-registrator
VERSION=$(shell cat VERSION)
DEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
PACKAGES = $(shell go list ./...)

deps:
	@echo "--> Installing build dependencies"
	@go get -d -v ./... $(DEPS)

docker:
	docker build -t $(NAME):$(VERSION) .

test:
	@echo "--> Running tests"
	@go test -v -cover ./...

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

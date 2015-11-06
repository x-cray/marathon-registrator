NAME=marathon-service-registrator
VERSION=$(shell cat VERSION)
PACKAGES=$(shell go list ./...)

deps:
	@echo "--> Installing dependencies"
	@go get -d -v -t ./...

test-deps:
	@which ginkgo 2>/dev/null ; if [ $$? -eq 1 ]; then \
		go get -u -v github.com/onsi/ginkgo/ginkgo; \
	fi

docker:
	@echo "--> Building docker image"
	@docker build -t $(NAME):$(VERSION) .

docker-dev:
	@echo "--> Building docker dev image"
	@docker build -f Dockerfile.dev -t $(NAME):dev .

test: test-deps
	@echo "--> Running tests"
	@ginkgo -r --randomizeAllSpecs --randomizeSuites --failOnPending --cover --trace --race

ci-test: test-deps
	@echo "--> Running CI tests"
	@ginkgo -r --randomizeAllSpecs --randomizeSuites --failOnPending --cover --trace --race --compilers=2

mocks:
	@echo "--> Generating mocks"
	@which mockgen 2>/dev/null ; if [ $$? -eq 1 ]; then \
		go get -u -v github.com/golang/mock/mockgen; \
	fi
	@mockgen -source=types/types.go -package=types -destination types/types_mocks.go
	@mockgen -source=marathon/address_resolver.go -package=marathon -destination marathon/address_resolver_mocks.go
	@mockgen -source=marathon/marathon_client.go -package=marathon -destination marathon/marathon_client_mocks.go

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

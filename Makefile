DOCKER_IMG_TAGGED = ellogroup/ello-golang-token-fetcher:latest
DOCKER_RUN = docker run --rm -it --platform linux/amd64 $(DOCKER_IMG_TAGGED)

.PHONY: build-format-test
build-format-test: build format test

.PHONY: build
build:
	docker build --platform linux/amd64 -t $(DOCKER_IMG_TAGGED) .
	$(DOCKER_RUN) go mod tidy

.PHONY: format
format:
	$(DOCKER_RUN) gofmt -w ./

.PHONY: test
test: static-tests unit-tests

.PHONY: static-tests
static-tests:
	$(DOCKER_RUN) golangci-lint run -v
	$(DOCKER_RUN) gosec ./...
	$(DOCKER_RUN) govulncheck ./...

.PHONY: unit-tests
unit-tests:
	$(DOCKER_RUN) go test -v -cover ./...
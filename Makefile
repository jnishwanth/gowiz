.PHONY: all verify tier1 tier2 tier3 build run clean install-hooks

all: build

verify:
	@./scripts/verify.sh

tier1:
	@gofmt -w .
	@go vet ./...
	@go test ./...

tier2:
	@go test -race ./...

tier3: build
	@./bin/gowiz --check

build:
	@mkdir -p bin
	@go build -o bin/gowiz .

run: build
	@./bin/gowiz

install-hooks:
	@./scripts/install-hooks.sh

clean:
	@rm -rf bin/

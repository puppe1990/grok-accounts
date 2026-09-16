GO ?= go
BIN := bin/grok-accounts

.PHONY: verify fmt-check vet test build install clean

verify: fmt-check vet test build

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt pendente em:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

build:
	$(GO) build -o $(BIN) ./cmd/grok-accounts

install:
	$(GO) install ./cmd/grok-accounts

clean:
	rm -rf bin

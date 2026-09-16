GO ?= go
PNPM ?= pnpm
BIN := bin/grok-accounts

.PHONY: verify deps fmt-check vet lint test build install clean

verify: fmt-check vet lint test build

deps:
	$(PNPM) install --frozen-lockfile

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt pendente em:"; echo "$$out"; exit 1; fi
	$(PNPM) exec prettier --check .

vet:
	$(GO) vet ./...

lint:
	golangci-lint run

test:
	$(GO) test ./...

build:
	$(GO) build -o $(BIN) ./cmd/grok-accounts

install:
	$(GO) install ./cmd/grok-accounts

clean:
	rm -rf bin

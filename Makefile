.PHONY: build test test-race clean lint fmt vet install-skill

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
  -X github.com/nfedorov/port_server/internal/version.Version=$(VERSION) \
  -X github.com/nfedorov/port_server/internal/version.Commit=$(COMMIT) \
  -X github.com/nfedorov/port_server/internal/version.Date=$(DATE)

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/port-server ./cmd/server
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/portctl ./cmd/portctl

test:
	go test ./...

test-race:
	go test -race ./...

lint: vet
	@which golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping (go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)"

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf bin/ dist/

install-skill:
	mkdir -p ~/.claude/skills/port-manager/references
	cp skill/port-manager/SKILL.md ~/.claude/skills/port-manager/SKILL.md
	cp skill/port-manager/references/WORKFLOW.md ~/.claude/skills/port-manager/references/WORKFLOW.md
	mkdir -p ~/.agents/skills/port-manager/references
	cp skill/port-manager/SKILL.md ~/.agents/skills/port-manager/SKILL.md
	cp skill/port-manager/references/WORKFLOW.md ~/.agents/skills/port-manager/references/WORKFLOW.md
	@echo "Skill installed to ~/.claude/skills/port-manager/ and ~/.agents/skills/port-manager/"

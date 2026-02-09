.PHONY: build test test-race clean lint fmt vet install-skill

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
  -X github.com/n3r/port-registry/internal/version.Version=$(VERSION) \
  -X github.com/n3r/port-registry/internal/version.Commit=$(COMMIT) \
  -X github.com/n3r/port-registry/internal/version.Date=$(DATE)

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/port-registry ./cmd/server
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
	mkdir -p ~/.claude/skills/port-registry/references
	cp skill/port-registry/SKILL.md ~/.claude/skills/port-registry/SKILL.md
	cp skill/port-registry/references/WORKFLOW.md ~/.claude/skills/port-registry/references/WORKFLOW.md
	mkdir -p ~/.agents/skills/port-registry/references
	cp skill/port-registry/SKILL.md ~/.agents/skills/port-registry/SKILL.md
	cp skill/port-registry/references/WORKFLOW.md ~/.agents/skills/port-registry/references/WORKFLOW.md
	@echo "Skill installed to ~/.claude/skills/port-registry/ and ~/.agents/skills/port-registry/"

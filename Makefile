.PHONY: build test clean install-skill

build:
	go build -o bin/port-server ./cmd/server
	go build -o bin/portctl ./cmd/portctl

test:
	go test ./...

clean:
	rm -rf bin/

install-skill:
	mkdir -p ~/.claude/skills/port-manager/references
	cp skill/port-manager/SKILL.md ~/.claude/skills/port-manager/SKILL.md
	cp skill/port-manager/references/WORKFLOW.md ~/.claude/skills/port-manager/references/WORKFLOW.md
	mkdir -p ~/.agents/skills/port-manager/references
	cp skill/port-manager/SKILL.md ~/.agents/skills/port-manager/SKILL.md
	cp skill/port-manager/references/WORKFLOW.md ~/.agents/skills/port-manager/references/WORKFLOW.md
	@echo "Skill installed to ~/.claude/skills/port-manager/ and ~/.agents/skills/port-manager/"

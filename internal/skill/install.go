package skill

import (
	"os"
	"path/filepath"

	skilldata "github.com/nfedorov/port_server/skill"
)

// Platform represents an agent platform that supports skills.
type Platform struct {
	Name string // e.g. "Claude Code"
	Dir  string // e.g. "/Users/x/.claude"
}

// InstallError records an error installing to a specific platform.
type InstallError struct {
	Platform Platform
	Err      error
}

// InstallResult summarizes the outcome of a skill install operation.
type InstallResult struct {
	Installed []Platform
	Skipped   []Platform
	Errors    []InstallError
}

// Install detects agent platforms and writes skill files to each.
// homeDir is the user's home directory; cwd is the current working directory.
func Install(homeDir, cwd string) InstallResult {
	var result InstallResult

	// Global platforms: install if the parent directory exists.
	globals := []Platform{
		{Name: "Claude Code", Dir: filepath.Join(homeDir, ".claude")},
		{Name: "OpenAI Codex", Dir: filepath.Join(homeDir, ".codex")},
		{Name: "Generic Agents", Dir: filepath.Join(homeDir, ".agents")},
	}

	for _, p := range globals {
		installPlatform(p, &result)
	}

	// Project-level: .claude/ in cwd (only if it already exists).
	if cwd != "" {
		projectDir := filepath.Join(cwd, ".claude")
		if info, err := os.Stat(projectDir); err == nil && info.IsDir() {
			p := Platform{Name: "Claude Code (project)", Dir: projectDir}
			installPlatform(p, &result)
		}
	}

	return result
}

func installPlatform(p Platform, result *InstallResult) {
	info, err := os.Stat(p.Dir)
	if err != nil || !info.IsDir() {
		result.Skipped = append(result.Skipped, p)
		return
	}

	destDir := filepath.Join(p.Dir, "skills", "port-manager", "references")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.Errors = append(result.Errors, InstallError{Platform: p, Err: err})
		return
	}

	skillPath := filepath.Join(p.Dir, "skills", "port-manager", "SKILL.md")
	if err := os.WriteFile(skillPath, skilldata.SkillMD, 0644); err != nil {
		result.Errors = append(result.Errors, InstallError{Platform: p, Err: err})
		return
	}

	workflowPath := filepath.Join(destDir, "WORKFLOW.md")
	if err := os.WriteFile(workflowPath, skilldata.WorkflowMD, 0644); err != nil {
		result.Errors = append(result.Errors, InstallError{Platform: p, Err: err})
		return
	}

	result.Installed = append(result.Installed, p)
}

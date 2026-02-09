package skill

import (
	"os"
	"path/filepath"

	skilldata "github.com/n3r/port-registry/skill"
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

// Install writes skill files to the appropriate platforms.
// When global is true, it installs to global platforms (~/.claude, ~/.codex, ~/.agents).
// When global is false, it installs to the project-local .claude/ in cwd (creating it if needed).
func Install(homeDir, cwd string, global bool) InstallResult {
	var result InstallResult

	if global {
		globals := []Platform{
			{Name: "Claude Code", Dir: filepath.Join(homeDir, ".claude")},
			{Name: "OpenAI Codex", Dir: filepath.Join(homeDir, ".codex")},
			{Name: "Generic Agents", Dir: filepath.Join(homeDir, ".agents")},
		}
		for _, p := range globals {
			installPlatform(p, &result)
		}
		return result
	}

	// Local install: .claude/ in cwd (create if needed).
	if cwd != "" {
		projectDir := filepath.Join(cwd, ".claude")
		p := Platform{Name: "Claude Code (project)", Dir: projectDir}
		installPlatformCreate(p, &result)
	}

	return result
}

// installPlatformCreate creates the platform directory if needed, then writes skill files.
func installPlatformCreate(p Platform, result *InstallResult) {
	destDir := filepath.Join(p.Dir, "skills", "port-registry", "references")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.Errors = append(result.Errors, InstallError{Platform: p, Err: err})
		return
	}
	writeSkillFiles(p, destDir, result)
}

// installPlatform writes skill files only if the platform directory already exists.
func installPlatform(p Platform, result *InstallResult) {
	info, err := os.Stat(p.Dir)
	if err != nil || !info.IsDir() {
		result.Skipped = append(result.Skipped, p)
		return
	}

	destDir := filepath.Join(p.Dir, "skills", "port-registry", "references")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.Errors = append(result.Errors, InstallError{Platform: p, Err: err})
		return
	}
	writeSkillFiles(p, destDir, result)
}

func writeSkillFiles(p Platform, destDir string, result *InstallResult) {
	skillPath := filepath.Join(p.Dir, "skills", "port-registry", "SKILL.md")
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

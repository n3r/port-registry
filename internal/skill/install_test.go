package skill

import (
	"os"
	"path/filepath"
	"testing"

	skilldata "github.com/nfedorov/port_server/skill"
)

func TestInstallGlobalDetectsAndWritesToPlatforms(t *testing.T) {
	home := t.TempDir()

	// Create .claude and .agents directories; skip .codex.
	os.Mkdir(filepath.Join(home, ".claude"), 0755)
	os.Mkdir(filepath.Join(home, ".agents"), 0755)

	result := Install(home, "", true)

	if len(result.Installed) != 2 {
		t.Fatalf("expected 2 installed, got %d", len(result.Installed))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(result.Skipped))
	}
	if result.Skipped[0].Name != "OpenAI Codex" {
		t.Errorf("expected skipped platform to be OpenAI Codex, got %s", result.Skipped[0].Name)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(result.Errors))
	}

	// Verify file contents for .claude.
	skillMD, err := os.ReadFile(filepath.Join(home, ".claude", "skills", "port-manager", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(skillMD) != string(skilldata.SkillMD) {
		t.Error("SKILL.md content mismatch")
	}

	workflowMD, err := os.ReadFile(filepath.Join(home, ".claude", "skills", "port-manager", "references", "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(workflowMD) != string(skilldata.WorkflowMD) {
		t.Error("WORKFLOW.md content mismatch")
	}

	// Verify files for .agents too.
	skillMD, err = os.ReadFile(filepath.Join(home, ".agents", "skills", "port-manager", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(skillMD) != string(skilldata.SkillMD) {
		t.Error(".agents SKILL.md content mismatch")
	}
}

func TestInstallLocalDefault(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()

	// Default install (local) should create .claude/ in cwd even if it doesn't exist.
	result := Install(home, cwd, false)

	if len(result.Installed) != 1 {
		t.Fatalf("expected 1 installed (project), got %d", len(result.Installed))
	}

	// Verify project-level install.
	if _, err := os.Stat(filepath.Join(cwd, ".claude", "skills", "port-manager", "SKILL.md")); err != nil {
		t.Error("project-level SKILL.md not found")
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude", "skills", "port-manager", "references", "WORKFLOW.md")); err != nil {
		t.Error("project-level WORKFLOW.md not found")
	}

	// Should NOT install globally.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills")); err == nil {
		t.Error("global install should not happen in local mode")
	}

	// Check that the platform has the right name.
	if result.Installed[0].Name != "Claude Code (project)" {
		t.Errorf("expected 'Claude Code (project)', got %s", result.Installed[0].Name)
	}
}

func TestInstallGlobalNoPlatforms(t *testing.T) {
	home := t.TempDir()

	result := Install(home, "", true)

	if len(result.Installed) != 0 {
		t.Fatalf("expected 0 installed, got %d", len(result.Installed))
	}
	if len(result.Skipped) != 3 {
		t.Fatalf("expected 3 skipped, got %d", len(result.Skipped))
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(result.Errors))
	}
}

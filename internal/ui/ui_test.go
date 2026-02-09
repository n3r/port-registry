package ui

import (
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	out := Success("done")
	if !strings.Contains(out, SymCheck) {
		t.Errorf("Success output should contain %q, got %q", SymCheck, out)
	}
	if !strings.Contains(out, "done") {
		t.Errorf("Success output should contain message, got %q", out)
	}
}

func TestError(t *testing.T) {
	out := Error("failed")
	if !strings.Contains(out, SymCross) {
		t.Errorf("Error output should contain %q, got %q", SymCross, out)
	}
	if !strings.Contains(out, "failed") {
		t.Errorf("Error output should contain message, got %q", out)
	}
}

func TestWarning(t *testing.T) {
	out := Warning("careful")
	if !strings.Contains(out, SymWarning) {
		t.Errorf("Warning output should contain %q, got %q", SymWarning, out)
	}
}

func TestInfo(t *testing.T) {
	out := Info("note")
	if !strings.Contains(out, SymBullet) {
		t.Errorf("Info output should contain %q, got %q", SymBullet, out)
	}
}

func TestTable(t *testing.T) {
	out := Table(
		[]string{"NAME", "VALUE"},
		[][]string{{"foo", "bar"}, {"baz", "qux"}},
	)
	if !strings.Contains(out, "NAME") {
		t.Errorf("Table should contain header, got %q", out)
	}
	if !strings.Contains(out, "foo") {
		t.Errorf("Table should contain data, got %q", out)
	}
	if !strings.Contains(out, "baz") {
		t.Errorf("Table should contain all rows, got %q", out)
	}
}

func TestFormattedHelpers(t *testing.T) {
	out := Successf("port %d allocated", 8080)
	if !strings.Contains(out, "8080") {
		t.Errorf("Successf should format args, got %q", out)
	}

	out = Errorf("failed: %s", "timeout")
	if !strings.Contains(out, "timeout") {
		t.Errorf("Errorf should format args, got %q", out)
	}
}

func TestSubtle(t *testing.T) {
	out := Subtle("(pid 123)")
	if !strings.Contains(out, "pid 123") {
		t.Errorf("Subtle should contain message, got %q", out)
	}
}

func TestBold(t *testing.T) {
	out := Bold("portctl")
	if !strings.Contains(out, "portctl") {
		t.Errorf("Bold should contain message, got %q", out)
	}
}

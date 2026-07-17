package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigAcceptsMissingProjectColor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("projects:\n- code: core\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConfig(path); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigAcceptsHexProjectColor(t *testing.T) {
	for _, color := range []string{"#0f0", "#22c55e"} {
		t.Run(color, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte("projects:\n- code: core\n  color: \""+color+"\"\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Projects[0].Color != color {
				t.Fatalf("color = %q, want %q", cfg.Projects[0].Color, color)
			}
		})
	}
}

func TestLoadConfigRejectsInvalidProjectColor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("projects:\n- code: core\n  color: green\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "projects[0].color must be #RGB or #RRGGBB") {
		t.Fatalf("expected color validation error, got %v", err)
	}
}

func TestParseHexColorExpandsShortForm(t *testing.T) {
	r, g, b, ok := ParseHexColor("#0f8")
	if !ok {
		t.Fatal("expected short hex color to parse")
	}
	if r != 0x00 || g != 0xff || b != 0x88 {
		t.Fatalf("rgb = %02x %02x %02x, want 00 ff 88", r, g, b)
	}
}

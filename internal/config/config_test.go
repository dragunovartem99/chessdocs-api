package config

import (
	"strings"
	"testing"
)

func complete() map[string]string {
	return map[string]string{
		"ALLOWED_ORIGIN":     "https://chessdocs.org",
		"GITHUB_TOKEN":       "ghp_example",
		"GITHUB_OWNER":       "dragunovartem99",
		"GITHUB_REPO":        "chessdocs",
		"GITHUB_BASE_BRANCH": "main",
		"PORT":               "8787",
	}
}

func environment(t *testing.T, values map[string]string) {
	t.Helper()
	for key, value := range values {
		t.Setenv(key, value)
	}
}

func TestLoad(t *testing.T) {
	environment(t, complete())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if cfg.Addr != ":8787" {
		t.Errorf("Addr = %q, want :8787", cfg.Addr)
	}
	if cfg.GitHub.Repo != "chessdocs" || cfg.GitHub.BaseBranch != "main" {
		t.Errorf("GitHub = %+v", cfg.GitHub)
	}
}

func TestLoadReportsEveryMissingVariableAtOnce(t *testing.T) {
	values := complete()
	values["GITHUB_TOKEN"] = ""
	values["PORT"] = ""
	environment(t, values)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil, want an error")
	}
	for _, key := range []string{"GITHUB_TOKEN", "PORT"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}

func TestLoadRejectsANonPort(t *testing.T) {
	values := complete()
	values["PORT"] = "http"
	environment(t, values)

	if _, err := Load(); err == nil {
		t.Fatal("Load() = nil, want an error")
	}
}

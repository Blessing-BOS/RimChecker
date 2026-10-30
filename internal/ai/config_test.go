package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigAndPrompt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	promptPath := filepath.Join(dir, "custom.md")
	if err := SavePrompt(promptPath, "自定义诊断规则"); err != nil {
		t.Fatal(err)
	}
	data := `{"base_url":"https://example.invalid/custom/v1/","model":"another-model","prompt_file":"custom.md"}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := ReadPrompt(c.PromptFile)
	if err != nil || prompt != "自定义诊断规则" || c.PromptFile != promptPath || c.TimeoutSeconds != 240 || c.MaxEvents != 20 {
		t.Fatalf("config/prompt mismatch: %v", err)
	}
	endpoint, _ := c.Endpoint()
	if endpoint != "https://example.invalid/custom/v1/chat/completions" {
		t.Fatal(endpoint)
	}
	if err := SavePrompt(promptPath, "修改后的提示词"); err != nil {
		t.Fatal(err)
	}
	if prompt, err := ReadPrompt(promptPath); err != nil || prompt != "修改后的提示词" {
		t.Fatal("prompt update failed")
	}
	for _, text := range []string{"", strings.Repeat("x", MaxPromptBytes+1), string([]byte{0xff})} {
		if err := SavePrompt(promptPath, text); err == nil {
			t.Fatal("invalid prompt accepted")
		}
	}
}

func TestInvalidConfig(t *testing.T) {
	for _, data := range []string{
		`null`, `{}`, `not json`,
		`{"base_url":"https://example.invalid/v1","model":"x","prompt_file":"p","api_key":"synthetic-secret"}`,
		`{"base_url":"https://example.invalid/v1","model":"x","prompt_file":"p","timeout_seconds":0}`,
		`{"base_url":"https://example.invalid/v1","model":"x","prompt_file":"p","max_events":0}`,
		`{"base_url":"https://example.invalid/v1","model":"x","prompt_file":"p","max_input_chars":10}`,
		`{"base_url":"https://example.invalid/v1","model":"x","prompt_file":"p"} {}`,
	} {
		path := filepath.Join(t.TempDir(), "invalid.json")
		_ = os.WriteFile(path, []byte(data), 0600)
		if _, err := LoadConfig(path); err == nil || strings.Contains(err.Error(), "synthetic-secret") {
			t.Fatalf("unsafe config error: %v", err)
		}
	}
	for _, base := range []string{"http://public.example/v1", "https://user:secret@example.invalid/v1", "https://example.invalid/v1?key=secret", "ftp://localhost/v1", "https://example.invalid/#secret"} {
		c := testConfig(base)
		if err := c.Validate(); err == nil {
			t.Fatal("unsafe URL accepted")
		}
	}
	if _, err := ReadPrompt(filepath.Join(t.TempDir(), "missing.md")); err == nil {
		t.Fatal("missing prompt accepted")
	}
}

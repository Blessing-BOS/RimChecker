package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAIUploadGatesAndPreview(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer synthetic-cli-key" {
			t.Error("missing synthetic credential")
		}
		fmt.Fprint(w, `{"choices":[{"message":{"content":"合成诊断结果"}}]}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "prompt.md")
	configPath := filepath.Join(dir, "ai.json")
	_ = os.WriteFile(promptPath, []byte("使用自定义中文规则"), 0600)
	config, _ := json.Marshal(map[string]any{"base_url": server.URL + "/v1", "model": "synthetic-model", "prompt_file": "prompt.md"})
	_ = os.WriteFile(configPath, config, 0600)
	logPath := filepath.Join(dir, "synthetic private log.log")
	_ = os.WriteFile(logPath, []byte("Error: C:\\Users\\Alice\\Player.log api_key=private123\n  at Synthetic.Run()\n"), 0600)
	args := []string{"-log", logPath, "-mode", "ai", "-config", configPath}
	t.Setenv("RIMCHECKER_API_KEY", "")
	var stdout, stderr bytes.Buffer
	if code := run(args, &stdout, &stderr); code != 2 || calls.Load() != 0 || stdout.Len() != 0 {
		t.Fatal("unconfirmed upload was not blocked")
	}
	stdout.Reset()
	stderr.Reset()
	if code := run(append(args, "-preview"), &stdout, &stderr); code != 0 || calls.Load() != 0 || !json.Valid(stdout.Bytes()) {
		t.Fatalf("preview: %d %s", code, &stderr)
	}
	if strings.Contains(stdout.String(), "Alice") || strings.Contains(stdout.String(), "private123") || !strings.Contains(stdout.String(), "使用自定义中文规则") {
		t.Fatal("preview sanitization or custom prompt failed")
	}
	stdout.Reset()
	stderr.Reset()
	if code := run(append(args, "-allow-upload"), &stdout, &stderr); code != 1 || calls.Load() != 0 || !strings.Contains(stderr.String(), "RIMCHECKER_API_KEY") {
		t.Fatal("missing key was accepted")
	}
	t.Setenv("RIMCHECKER_API_KEY", "synthetic-cli-key")
	stdout.Reset()
	stderr.Reset()
	if code := run(append(args, "-preview", "-allow-upload"), &stdout, &stderr); code != 0 || calls.Load() != 0 {
		t.Fatal("preview precedence failed")
	}
	stdout.Reset()
	stderr.Reset()
	if code := run(append(args, "-allow-upload"), &stdout, &stderr); code != 0 || calls.Load() != 1 || strings.TrimSpace(stdout.String()) != "合成诊断结果" {
		t.Fatalf("mock CLI: %d %s", code, &stderr)
	}
}

func TestAIMissingConfigAndInvalidModeOptions(t *testing.T) {
	for _, args := range [][]string{
		{"-log", "unused.log", "-mode", "ai", "-preview", "-config", filepath.Join(t.TempDir(), "missing.json")},
		{"-log", "unused.log", "-mode", "scan", "-allow-upload"},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatal("invalid AI options accepted")
		}
	}
}

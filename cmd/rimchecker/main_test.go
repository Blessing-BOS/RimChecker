package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		code    int
		message string
	}{
		{"help", []string{"-help"}, 0, "用法"},
		{"missing flag", nil, 2, "请通过 -log"},
		{"missing value", []string{"-log"}, 2, "flag needs an argument"},
		{"unknown flag", []string{"-unknown"}, 2, "flag provided but not defined"},
		{"extra argument", []string{"-log", "unused.log", "extra"}, 2, "不接受额外的位置参数"},
		{"missing file", []string{"-log", filepath.Join(t.TempDir(), "missing.log")}, 1, "扫描失败"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tt.args, &stdout, &stderr); code != tt.code {
				t.Fatalf("exit code=%d, want %d; stderr=%s", code, tt.code, &stderr)
			}
			if !strings.Contains(stderr.String(), tt.message) {
				t.Fatalf("stderr=%q, want %q", stderr.String(), tt.message)
			}
			if stdout.Len() != 0 {
				t.Fatalf("unexpected report: %s", &stdout)
			}
		})
	}
}

func TestRunPreviewAndSpaceInPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic sample with spaces.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("Error: synthetic\n", 12)), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-log", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code=%d stderr=%s", code, &stderr)
	}
	output := stdout.String()
	for _, want := range []string{"候选异常行数：12", "总行数：12", "  10 | Error |", "另有 2 条候选异常行未展示"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, "  11 |") || strings.Count(output, " | Error | ") != 10 {
		t.Errorf("preview must contain exactly 10 rows: %s", output)
	}
}

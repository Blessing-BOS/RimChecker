package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"rimchecker/internal/logparse"
	"rimchecker/internal/logscan"
)

func TestParseJSON(t *testing.T) {
	tests := []struct {
		file   string
		format logparse.Format
		count  int64
	}{
		{"sample.log", logparse.FormatLegacy, 6},
		{"legacy-synthetic.log", logparse.FormatLegacy, 4},
		{"waylith-synthetic.log", logparse.FormatWaylith, 5},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			path := filepath.Join("..", "..", "testdata", "fixtures", tt.file)
			if code := run([]string{"-log", path, "-mode", "parse", "-format", "json"}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
				t.Fatalf("code=%d stderr=%s", code, &stderr)
			}
			if !json.Valid(stdout.Bytes()) {
				t.Fatalf("invalid JSON: %s", &stdout)
			}
			decoder := json.NewDecoder(&stdout)
			var result logparse.Result
			if err := decoder.Decode(&result); err != nil {
				t.Fatal(err)
			}
			if result.Format != tt.format || result.DiagnosticCount != tt.count || len(result.Events) == 0 || len(result.Diagnostics) == 0 {
				t.Fatalf("incorrect JSON fields: %+v", result)
			}
			if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
				t.Fatalf("extra stdout content: %v", err)
			}
		})
	}
}

func TestParseText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	path := filepath.Join("..", "..", "testdata", "fixtures", "legacy-synthetic.log")
	if code := run([]string{"-log", path, "-mode", "parse"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, &stderr)
	}
	for _, want := range []string{"格式：Legacy", "错误/警告事件数：4", "去重后诊断组数：3", "发生次数：2", "首次出现：第 8–11 行", "末次出现：第 15–18 行", "合成空引用", "Synthetic.Mod.Tick"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("missing %q: %s", want, &stdout)
		}
	}
}

func TestExplicitScanMatchesDefault(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "fixtures", "sample.log")
	var implicit, explicit, stderr bytes.Buffer
	if code := run([]string{"-log", path}, &implicit, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if code := run([]string{"-log", path, "-mode", "scan"}, &explicit, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if implicit.String() != explicit.String() {
		t.Fatal("explicit scan changed the original default report")
	}
	var output bytes.Buffer
	if code := run([]string{"-log", path, "-format", "json"}, &output, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	var result logscan.Result
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.TotalLines != 11 || result.CandidateCount != 6 {
		t.Fatalf("unexpected scan JSON: %+v", result)
	}
}

func TestParseArgumentAndFileErrors(t *testing.T) {
	tests := []struct {
		args    []string
		code    int
		message string
	}{
		{[]string{"-log", "unused.log", "-mode", "invalid", "-format", "json"}, 2, "参数错误"},
		{[]string{"-log", "unused.log", "-format", "yaml"}, 2, "参数错误"},
		{[]string{"-log", filepath.Join(t.TempDir(), "missing.log"), "-mode", "parse", "-format", "json"}, 1, "解析失败"},
	}
	for _, tt := range tests {
		var stdout, stderr bytes.Buffer
		if code := run(tt.args, &stdout, &stderr); code != tt.code || stdout.Len() != 0 || !strings.Contains(stderr.String(), tt.message) {
			t.Errorf("args=%v code=%d stdout=%s stderr=%s", tt.args, code, &stdout, &stderr)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("synthetic output failure") }

func TestJSONWriteFailure(t *testing.T) {
	var stderr bytes.Buffer
	if code := writeJSON(map[string]string{"message": "synthetic"}, failingWriter{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "JSON 输出失败") {
		t.Fatalf("code=%d stderr=%s", code, &stderr)
	}
}

package logscan

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// All fixtures here are synthetic; none are real RimWorld logs.
func TestScan(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		lines      int64
		counts     TypeCounts
		candidates []CandidateLine
	}{
		{name: "normal log", content: "Starting\nLoading\nDone\n", lines: 3},
		{name: "empty file"},
		{name: "blank lines", content: "\n\n", lines: 2},
		{
			name:    "multiple keyword lines and final unterminated line",
			content: "Starting\nError: unavailable\nException: failed\nWarning: missing\nDone",
			lines:   5, counts: TypeCounts{Exception: 1, Error: 1, Warning: 1},
			candidates: []CandidateLine{
				{2, TypeError, "Error: unavailable"},
				{3, TypeException, "Exception: failed"},
				{4, TypeWarning, "Warning: missing"},
			},
		},
		{
			name: "Windows CRLF", content: "Starting\r\n\r\nWarning: missing\r\nError: failed\r\n",
			lines: 4, counts: TypeCounts{Error: 1, Warning: 1},
			candidates: []CandidateLine{{3, TypeWarning, "Warning: missing"}, {4, TypeError, "Error: failed"}},
		},
		{
			name: "case insensitive", content: "eRrOr\nEXCEPTION\nwArNiNg\n",
			lines: 3, counts: TypeCounts{Exception: 1, Error: 1, Warning: 1},
			candidates: []CandidateLine{{1, TypeError, "eRrOr"}, {2, TypeException, "EXCEPTION"}, {3, TypeWarning, "wArNiNg"}},
		},
		{
			name:    "overlapping and repeated keywords counted once",
			content: "Warning Error Exception Error\nWarning Error\nWarning Warning\n",
			lines:   3, counts: TypeCounts{Exception: 1, Error: 1, Warning: 1},
			candidates: []CandidateLine{
				{1, TypeException, "Warning Error Exception Error"},
				{2, TypeError, "Warning Error"},
				{3, TypeWarning, "Warning Warning"},
			},
		},
		{
			name: "substring match is preliminary", content: "No errors observed\nNullReferenceException\n",
			lines: 2, counts: TypeCounts{Error: 1, Exception: 1},
			candidates: []CandidateLine{{1, TypeError, "No errors observed"}, {2, TypeException, "NullReferenceException"}},
		},
		{
			name:    "longer than 64 KiB with keyword at end",
			content: strings.Repeat("x", 128*1024) + "Error\nWarning: next line",
			lines:   2, counts: TypeCounts{Error: 1, Warning: 1},
			candidates: []CandidateLine{{1, TypeError, strings.Repeat("x", SummaryRuneLimit) + "…"}, {2, TypeWarning, "Warning: next line"}},
		},
		{
			name:    "long final line and Unicode summary",
			content: "Exception " + strings.Repeat("合成", 40000),
			lines:   1, counts: TypeCounts{Exception: 1},
			candidates: []CandidateLine{{1, TypeException, "Exception " + strings.Repeat("合成", 95) + "…"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeLog(t, tt.content)
			got, err := Scan(path)
			if err != nil {
				t.Fatal(err)
			}
			if got.FilePath != path || got.FileSize != int64(len(tt.content)) || got.TotalLines != tt.lines {
				t.Fatalf("unexpected metadata: path=%q size=%d lines=%d", got.FilePath, got.FileSize, got.TotalLines)
			}
			if got.Counts != tt.counts || got.CandidateCount != int64(len(tt.candidates)) {
				t.Fatalf("counts=%+v total=%d; want %+v total=%d", got.Counts, got.CandidateCount, tt.counts, len(tt.candidates))
			}
			if len(got.Candidates) != len(tt.candidates) {
				t.Fatalf("got %d candidates, want %d", len(got.Candidates), len(tt.candidates))
			}
			for i, candidate := range got.Candidates {
				if !reflect.DeepEqual(candidate, tt.candidates[i]) {
					t.Errorf("candidate %d = %+v; want %+v", i, candidate, tt.candidates[i])
				}
				if !utf8.ValidString(candidate.Summary) {
					t.Errorf("candidate %d summary is not valid UTF-8", i)
				}
			}
		})
	}
}

func TestScanMissingPath(t *testing.T) {
	result, err := Scan(filepath.Join(t.TempDir(), "missing.log"))
	if result != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("result=%v err=%v; want nil result and missing-file error", result, err)
	}
}

func TestScanDirectory(t *testing.T) {
	if result, err := Scan(t.TempDir()); err == nil || result != nil {
		t.Fatalf("result=%v err=%v; directory must be rejected", result, err)
	}
}

func TestScanUnreadableFile(t *testing.T) {
	path := writeLog(t, "Error: synthetic\n")
	if err := os.Chmod(path, 0); err != nil {
		t.Skipf("cannot change file permissions: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0600); err != nil {
			t.Errorf("restore permissions: %v", err)
		}
	})
	// Windows permission semantics and privileged accounts may still allow reads.
	if probe, err := os.Open(path); err == nil {
		probe.Close()
		t.Skip("this environment permits reads despite mode 000; permission denial cannot be reproduced")
	}
	if result, err := Scan(path); err == nil || result != nil {
		t.Fatalf("result=%v err=%v; unreadable file must fail", result, err)
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestScanReaderFailure(t *testing.T) {
	want := errors.New("synthetic read failure")
	if err := scanReader(failingReader{want}, &Result{}); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func writeLog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "synthetic log with spaces.log")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

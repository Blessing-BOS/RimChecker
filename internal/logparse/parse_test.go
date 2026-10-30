package logparse

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Every file in testdata/fixtures and every input below is synthetic.
func TestSyntheticFixtures(t *testing.T) {
	tests := []struct {
		name                                     string
		format                                   Format
		lines                                    int64
		events                                   int
		diagnostics                              int64
		groups                                   int
		firstStart, firstEnd, lastStart, lastEnd int64
	}{
		{"legacy", FormatLegacy, 22, 7, 4, 3, 8, 11, 15, 18},
		{"waylith", FormatWaylith, 19, 8, 5, 4, 5, 7, 10, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "fixtures", tt.name+"-synthetic.log")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			result, err := Parse(path)
			if err != nil {
				t.Fatal(err)
			}
			if result.FilePath != path || result.FileSize != int64(len(data)) || result.Format != tt.format || result.TotalLines != tt.lines {
				t.Fatalf("metadata mismatch: %+v", result)
			}
			if len(result.Events) != tt.events || result.DiagnosticCount != tt.diagnostics || len(result.Diagnostics) != tt.groups {
				t.Fatalf("events=%d diagnostic_count=%d groups=%d", len(result.Events), result.DiagnosticCount, len(result.Diagnostics))
			}
			group := result.Diagnostics[0]
			if group.Count != 2 || group.FirstStartLine != tt.firstStart || group.FirstEndLine != tt.firstEnd || group.LastStartLine != tt.lastStart || group.LastEndLine != tt.lastEnd {
				t.Fatalf("unexpected first group: %+v", group)
			}
			if len(group.Event.Stack) < 2 || !strings.Contains(group.Event.Message, "合成空引用") {
				t.Fatalf("missing complete message/stack: %+v", group.Event)
			}
			if tt.format == FormatWaylith && (group.Event.Timestamp != "2026-10-30 12:00:01.002" || group.Event.Source != "Verse") {
				t.Fatalf("missing Waylith metadata: %+v", group.Event)
			}
			assertRawCoverage(t, result, string(data))
		})
	}
}

func TestParseBoundaries(t *testing.T) {
	tests := []struct {
		name, input string
		levels      []Level
		spans       [][2]int64
		stackSizes  []int
		format      Format
	}{
		{"consecutive exceptions", "System.NullReferenceException: one\n  at Synthetic.ErrorHandler()\nSystem.InvalidOperationException: two\n  at Synthetic.Other()",
			[]Level{LevelError, LevelError}, [][2]int64{{1, 2}, {3, 4}}, []int{1, 1}, FormatLegacy},
		{"next unmarked message owns its stack", "Error: first\nVerse.Log:Error (string)\nUnmarked normal message about Exception\nUnityEngine.StackTraceUtility:ExtractStackTrace ()\nVerse.Log:Message (string)\n",
			[]Level{LevelError, LevelInfo}, [][2]int64{{1, 2}, {3, 5}}, []int{1, 2}, FormatLegacy},
		{"ordinary body inferred from logger", "Missing synthetic def\nUnityEngine.StackTraceUtility:ExtractStackTrace ()\nVerse.Log:Warning (string)\n",
			[]Level{LevelWarning}, [][2]int64{{1, 3}}, []int{2}, FormatLegacy},
		{"info contains exception and Error stack caller", "INFO: Exception is a registered handler\nVerse.Log:Error (string)\n",
			[]Level{LevelInfo}, [][2]int64{{1, 2}}, []int{1}, FormatLegacy},
		{"unprefixed info refined by logger", "Exception handlers registered\nVerse.Log:Message (string)\nVerse.Log:Error (string)\n",
			[]Level{LevelInfo}, [][2]int64{{1, 3}}, []int{2}, FormatLegacy},
		{"waylith info ends error and takes its own stack", "2026-10-30 01:02:03.004 ERROR [Verse] failure\nat Synthetic.Fail()\n2026-10-30 01:02:04.005 INFO [Verse] Exception word only\nat Synthetic.ErrorReporter()\n",
			[]Level{LevelError, LevelInfo}, [][2]int64{{1, 2}, {3, 4}}, []int{1, 1}, FormatWaylith},
		{"multiline and indented inner exception", "Error: 外层异常\n包含中文的详细原因\n  System.InnerException: 内部原因\n  at Synthetic.Run()\n",
			[]Level{LevelError}, [][2]int64{{1, 4}}, []int{1}, FormatLegacy},
		{"incomplete stack at EOF", "Error: incomplete\n  at Synthetic.Call(",
			[]Level{LevelError}, [][2]int64{{1, 2}}, []int{1}, FormatLegacy},
		{"error without stack", "Error: title only", []Level{LevelError}, [][2]int64{{1, 1}}, []int{0}, FormatLegacy},
		{"unknown preserves keywords", "synthetic prose contains Error and Exception\nextra details\n", []Level{LevelUnknown}, [][2]int64{{1, 2}}, []int{0}, FormatUnknown},
		{"orphan stack", "  at Synthetic.ErrorMethod()\n", []Level{LevelUnknown}, [][2]int64{{1, 1}}, []int{1}, FormatUnknown},
		{"arbitrary Log.Error is not a Verse emitter", "unmarked ordinary text\nat Synthetic.Log.Error()\n", []Level{LevelUnknown}, [][2]int64{{1, 2}}, []int{1}, FormatUnknown},
		{"warning followed by tab", "Warning\tsynthetic\n", []Level{LevelWarning}, [][2]int64{{1, 1}}, []int{0}, FormatLegacy},
		{"blank file lines", "\n\n", []Level{LevelUnknown}, [][2]int64{{1, 2}}, []int{0}, FormatUnknown},
		{"mixed formats", "Warning: old syntax\n2026-10-30 01:02:03 ERROR [Verse] new syntax\n", []Level{LevelWarning, LevelError}, [][2]int64{{1, 1}, {2, 2}}, []int{0, 0}, FormatUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseText(t, tt.input)
			if len(result.Events) != len(tt.levels) || result.Format != tt.format {
				t.Fatalf("events=%d format=%s; want events=%d format=%s", len(result.Events), result.Format, len(tt.levels), tt.format)
			}
			for i, event := range result.Events {
				if event.Level != tt.levels[i] || event.StartLine != tt.spans[i][0] || event.EndLine != tt.spans[i][1] || len(event.Stack) != tt.stackSizes[i] {
					t.Errorf("event %d: %+v", i, event)
				}
				if event.Level == LevelInfo && event.IsDiagnostic() {
					t.Error("INFO counted as diagnostic")
				}
			}
			assertRawCoverage(t, result, tt.input)
		})
	}
}

func TestEmpty(t *testing.T) {
	result := parseText(t, "")
	if result.TotalLines != 0 || result.Format != FormatUnknown || len(result.Events) != 0 || len(result.Diagnostics) != 0 || result.DiagnosticCount != 0 {
		t.Fatalf("unexpected empty result: %+v", result)
	}
	if result.Events == nil || result.Diagnostics == nil {
		t.Fatal("JSON collections should be arrays, not null")
	}
}

func TestLineEndingsAndBOM(t *testing.T) {
	lf := "\uFEFF2026-10-30T01:02:03.004+08:00 ERROR [测试模块] 合成异常\n详细说明\n  at Synthetic.Run()\n2026-10-30T01:02:04Z INFO [测试模块] done"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	a, b := parseText(t, lf), parseText(t, crlf)
	assertRawCoverage(t, a, lf)
	assertRawCoverage(t, b, crlf)
	if a.TotalLines != 4 || b.TotalLines != 4 || a.Format != FormatWaylith || b.Format != FormatWaylith {
		t.Fatal("incorrect format or line counts")
	}
	for i := range a.Events {
		a.Events[i].RawText, b.Events[i].RawText = "", ""
		if !reflect.DeepEqual(a.Events[i], b.Events[i]) {
			t.Fatalf("newline-dependent parsing: %+v vs %+v", a.Events[i], b.Events[i])
		}
	}
	if a.Events[0].Timestamp != "2026-10-30T01:02:03.004+08:00" || a.Events[0].Source != "测试模块" || a.Events[0].Message != "合成异常\n详细说明" {
		t.Fatalf("unexpected structured fields: %+v", a.Events[0])
	}
}

func TestUncertainContentIsRetained(t *testing.T) {
	input := "Error: first line\npossibly another unmarked record\nat Synthetic.Run()\n"
	result := parseText(t, input)
	if !result.Events[0].BoundaryUncertain || !strings.Contains(result.Events[0].Message, "possibly another") {
		t.Fatalf("missing conservative context: %+v", result.Events[0])
	}
	assertRawCoverage(t, result, input)
}

func TestLongLine(t *testing.T) {
	message := "合成" + strings.Repeat("x", 128*1024)
	result := parseText(t, "Error: "+message+"\nat Synthetic.Run()\n")
	if result.TotalLines != 2 || result.Events[0].Message != message || len(result.Events[0].Stack) != 1 {
		t.Fatal("long line was lost or truncated")
	}
}

func TestParseFileErrors(t *testing.T) {
	if result, err := Parse(filepath.Join(t.TempDir(), "missing.log")); result != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("result=%v err=%v", result, err)
	}
	if result, err := Parse(t.TempDir()); result != nil || err == nil {
		t.Fatalf("directory accepted: result=%v err=%v", result, err)
	}
}

func TestParseFileWithSpaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic log with spaces.log")
	input := "Error: 合成读取测试\r\n  at Synthetic.Run()"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.FilePath != path || result.FileSize != int64(len(input)) || result.DiagnosticCount != 1 {
		t.Fatalf("incorrect file metadata: %+v", result)
	}
	assertRawCoverage(t, result, input)
}

type brokenReader struct{ err error }

func (r brokenReader) Read([]byte) (int, error) { return 0, r.err }

func TestReaderError(t *testing.T) {
	want := errors.New("synthetic failure")
	if result, err := parseReader(brokenReader{want}); result != nil || !errors.Is(err, want) {
		t.Fatalf("result=%v err=%v", result, err)
	}
}

func parseText(t *testing.T, input string) *Result {
	t.Helper()
	result, err := parseReader(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertRawCoverage(t *testing.T, result *Result, input string) {
	t.Helper()
	var raw strings.Builder
	nextLine := int64(1)
	for _, event := range result.Events {
		if event.StartLine != nextLine || event.EndLine < event.StartLine {
			t.Fatalf("noncontiguous lines: %+v", event)
		}
		raw.WriteString(event.RawText)
		nextLine = event.EndLine + 1
	}
	if raw.String() != input || nextLine != result.TotalLines+1 {
		t.Fatal("raw input lost, reordered or assigned incorrect line ranges")
	}
}

package logparse

import (
	"fmt"
	"strings"
	"testing"
)

func TestDeduplicateRepeatedEvents(t *testing.T) {
	var input strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&input, "2026-10-30 01:02:03.%03d ERROR [Verse] System.NullReferenceException: synthetic pawn 42\nat Synthetic.Tick() in C:\\Mods\\Example\\Pawn.cs:120\n", i)
	}
	result := parseText(t, input.String())
	if result.DiagnosticCount != 100 || len(result.Diagnostics) != 1 {
		t.Fatalf("count=%d groups=%d", result.DiagnosticCount, len(result.Diagnostics))
	}
	group := result.Diagnostics[0]
	if group.Count != 100 || group.FirstStartLine != 1 || group.FirstEndLine != 2 || group.LastStartLine != 199 || group.LastEndLine != 200 {
		t.Fatalf("incorrect occurrence ranges: %+v", group)
	}
	if group.Event.Timestamp != "2026-10-30 01:02:03.000" || group.Event.RawText != result.Events[0].RawText || len(group.Event.Stack) != 1 {
		t.Fatal("first representative was not retained")
	}
	assertRawCoverage(t, result, input.String())
}

func TestDeduplicatePreservesMeaningfulDifferences(t *testing.T) {
	base := DiagnosticEvent{Level: LevelError, Source: "Verse", Message: "SyntheticException: pawn 42 at C:\\Mods\\ExampleA\\def.xml", Stack: []string{"at Synthetic.Tick() in Synthetic.dll:120"}, StartLine: 1, EndLine: 2}
	tests := []struct {
		name   string
		change func(*DiagnosticEvent)
	}{
		{"different stack method", func(e *DiagnosticEvent) { e.Stack = []string{"at Synthetic.OtherTick() in Synthetic.dll:120"} }},
		{"different stack number", func(e *DiagnosticEvent) { e.Stack = []string{"at Synthetic.Tick() in Synthetic.dll:121"} }},
		{"different body number", func(e *DiagnosticEvent) { e.Message = strings.Replace(e.Message, "42", "43", 1) }},
		{"different path", func(e *DiagnosticEvent) { e.Message = strings.Replace(e.Message, "C:", "D:", 1) }},
		{"different mod identifier", func(e *DiagnosticEvent) { e.Message = strings.Replace(e.Message, "ExampleA", "ExampleB", 1) }},
		{"different source", func(e *DiagnosticEvent) { e.Source = "OtherModule" }},
		{"different level", func(e *DiagnosticEvent) { e.Level = LevelWarning }},
		{"missing stack", func(e *DiagnosticEvent) { e.Stack = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			other := base
			other.StartLine, other.EndLine = 3, 4
			tt.change(&other)
			groups := Deduplicate([]DiagnosticEvent{base, other, base})
			if len(groups) != 2 || groups[0].Count != 2 || groups[1].Count != 1 {
				t.Fatalf("incorrect merge: %+v", groups)
			}
		})
	}
}

func TestDeduplicateIgnoresMetadataAndFrameIndentation(t *testing.T) {
	input := "Error: synthetic\n  at Synthetic.Tick()\n[Ref ABC1]\n(Filename: synthetic.cs Line: 1)\n\nError: synthetic\nat Synthetic.Tick()\n[Ref DEF2]\n(Filename: synthetic.cs Line: 2)\n"
	result := parseText(t, input)
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Count != 2 {
		t.Fatalf("metadata prevented merge: %+v", result.Diagnostics)
	}
	assertRawCoverage(t, result, input)
}

func TestDeduplicateExcludesInfoAndUnknown(t *testing.T) {
	groups := Deduplicate([]DiagnosticEvent{
		{Level: LevelInfo, Message: "Exception is mentioned"},
		{Level: LevelUnknown, Message: "Error is mentioned"},
	})
	if len(groups) != 0 {
		t.Fatalf("non-diagnostic records grouped: %+v", groups)
	}
}

func TestReferenceLikeBodyIsNotRemoved(t *testing.T) {
	input := "Error: synthetic\n[Ref ExampleA] meaningful body 100\nat Synthetic.Run()\nError: synthetic\n[Ref ExampleB] meaningful body 200\nat Synthetic.Run()\n"
	result := parseText(t, input)
	if len(result.Diagnostics) != 2 || !strings.Contains(result.Events[0].Message, "meaningful body 100") {
		t.Fatal("reference-like message content was stripped or incorrectly merged")
	}
	assertRawCoverage(t, result, input)
}

func TestHeaderMessageResemblingMetadataIsPreserved(t *testing.T) {
	input := "Error: [Ref ABC1]\nat Synthetic.Run()\nError: [Ref DEF2]\nat Synthetic.Run()\n"
	result := parseText(t, input)
	if len(result.Diagnostics) != 2 || result.Events[0].Message != "[Ref ABC1]" || result.Events[1].Message != "[Ref DEF2]" {
		t.Fatal("header body was mistaken for standalone record metadata")
	}
	assertRawCoverage(t, result, input)
}

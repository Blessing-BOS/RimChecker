package logparse

import "strings"

// Deduplicate groups only diagnostic records, in first-occurrence order. The
// key excludes timestamp, raw prefix and line numbers, but preserves source,
// level, message, all stack frames, numbers, paths and identifiers exactly.
func Deduplicate(events []DiagnosticEvent) []DiagnosticGroup {
	type key struct {
		level   Level
		source  string
		message string
		stack   string
	}
	groups := []DiagnosticGroup{}
	indices := make(map[key]int)
	for _, event := range events {
		if !event.IsDiagnostic() {
			continue
		}
		frames := make([]string, len(event.Stack))
		for i, frame := range event.Stack {
			frames[i] = strings.TrimSpace(frame)
		}
		fingerprint := key{event.Level, event.Source, event.Message, strings.Join(frames, "\n")}
		if index, ok := indices[fingerprint]; ok {
			group := &groups[index]
			group.Count++
			group.LastStartLine, group.LastEndLine = event.StartLine, event.EndLine
			continue
		}
		indices[fingerprint] = len(groups)
		groups = append(groups, DiagnosticGroup{
			Event: event, Count: 1,
			FirstStartLine: event.StartLine, FirstEndLine: event.EndLine,
			LastStartLine: event.StartLine, LastEndLine: event.EndLine,
		})
	}
	return groups
}

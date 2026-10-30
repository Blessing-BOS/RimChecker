// Package logparse extracts records using a limited set of observed RimWorld
// text log conventions. Format names describe syntax, not verified game versions.
package logparse

type Format string

const (
	FormatUnknown Format = "Unknown"
	FormatLegacy  Format = "Legacy"
	FormatWaylith Format = "Waylith"
)

type Level string

const (
	LevelUnknown Level = "UNKNOWN"
	LevelTrace   Level = "TRACE"
	LevelDebug   Level = "DEBUG"
	LevelInfo    Level = "INFO"
	LevelWarning Level = "WARNING"
	LevelError   Level = "ERROR"
	LevelFatal   Level = "FATAL"
)

// DiagnosticEvent is also used for INFO and unclassified records so that every
// input line remains available. RawText includes the original line terminators.
// Message includes non-stack continuation text; Stack contains complete frames.
type DiagnosticEvent struct {
	Format            Format   `json:"format"`
	Level             Level    `json:"level"`
	Title             string   `json:"title"`
	Message           string   `json:"message"`
	Stack             []string `json:"stack"`
	RawText           string   `json:"raw_text"`
	StartLine         int64    `json:"start_line"`
	EndLine           int64    `json:"end_line"`
	Timestamp         string   `json:"timestamp"`
	Source            string   `json:"source"`
	BoundaryUncertain bool     `json:"boundary_uncertain"`
}

func (e DiagnosticEvent) IsDiagnostic() bool {
	return e.Level == LevelError || e.Level == LevelWarning || e.Level == LevelFatal
}

// DiagnosticGroup keeps the first complete event as the representative.
// First/Last refer to occurrence order, and both ends of each range are inclusive.
type DiagnosticGroup struct {
	Event          DiagnosticEvent `json:"event"`
	Count          int64           `json:"count"`
	FirstStartLine int64           `json:"first_start_line"`
	FirstEndLine   int64           `json:"first_end_line"`
	LastStartLine  int64           `json:"last_start_line"`
	LastEndLine    int64           `json:"last_end_line"`
}

type Result struct {
	FilePath        string            `json:"file_path"`
	FileSize        int64             `json:"file_size"`
	TotalLines      int64             `json:"total_lines"`
	Format          Format            `json:"format"`
	Events          []DiagnosticEvent `json:"events"`
	DiagnosticCount int64             `json:"diagnostic_count"`
	Diagnostics     []DiagnosticGroup `json:"diagnostics"`
}

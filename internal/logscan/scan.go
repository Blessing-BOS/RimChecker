// Package logscan performs a preliminary keyword scan, not RimWorld-specific parsing.
package logscan

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// SummaryRuneLimit bounds each retained summary without cutting UTF-8 characters.
const SummaryRuneLimit = 200

type CandidateType string

const (
	TypeException CandidateType = "Exception"
	TypeError     CandidateType = "Error"
	TypeWarning   CandidateType = "Warning"
)

// CandidateLine describes one matching line. LineNumber starts at 1.
type CandidateLine struct {
	LineNumber int64         `json:"line_number"`
	Type       CandidateType `json:"type"`
	Summary    string        `json:"summary"`
}

type TypeCounts struct {
	Exception int64 `json:"exception"`
	Error     int64 `json:"error"`
	Warning   int64 `json:"warning"`
}

// Result retains metadata and bounded summaries for all candidate lines.
// FileSize is the size in bytes observed when the file was opened.
type Result struct {
	FilePath       string          `json:"file_path"`
	FileSize       int64           `json:"file_size"`
	TotalLines     int64           `json:"total_lines"`
	CandidateCount int64           `json:"candidate_count"`
	Counts         TypeCounts      `json:"counts"`
	Candidates     []CandidateLine `json:"candidates"`
}

// Scan reads an explicitly supplied regular file incrementally. It accepts LF
// and CRLF; a nonempty final line without a newline also counts as one line.
func Scan(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件 %q 失败: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("读取日志文件信息 %q 失败: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("日志路径 %q 不是普通文件", path)
	}

	result := &Result{FilePath: path, FileSize: info.Size(), Candidates: []CandidateLine{}}
	if err := scanReader(f, result); err != nil {
		return nil, fmt.Errorf("读取日志文件 %q 失败: %w", path, err)
	}
	return result, nil
}

func scanReader(input io.Reader, result *Result) error {
	reader := bufio.NewReader(input)
	for {
		// ReadString has no Scanner-style 64 KiB token limit. Only the current
		// complete line and candidate summaries are kept, not the entire file.
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if len(line) > 0 {
			result.TotalLines++
			line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			kind := classify(line)
			if kind != "" {
				result.CandidateCount++
				switch kind {
				case TypeException:
					result.Counts.Exception++
				case TypeError:
					result.Counts.Error++
				case TypeWarning:
					result.Counts.Warning++
				}
				result.Candidates = append(result.Candidates, CandidateLine{
					LineNumber: result.TotalLines,
					Type:       kind,
					Summary:    summarize(line),
				})
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
}

// A line gets exactly one type, using Exception > Error > Warning precedence.
// Substring matches are deliberate and do not establish a real error or cause.
func classify(line string) CandidateType {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "exception"):
		return TypeException
	case strings.Contains(lower, "error"):
		return TypeError
	case strings.Contains(lower, "warning"):
		return TypeWarning
	default:
		return ""
	}
}

func summarize(line string) string {
	count := 0
	for index := range line {
		if count == SummaryRuneLimit {
			// Clone prevents a short substring from retaining a very long line.
			return strings.ReplaceAll(strings.Clone(line[:index]), "\t", " ") + "…"
		}
		count++
	}
	return strings.ReplaceAll(strings.Clone(line), "\t", " ")
}

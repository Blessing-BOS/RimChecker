package logparse

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Parse reads one explicitly supplied regular file. Input is streamed, but all
// records and their complete raw text are retained in the returned result.
func Parse(path string) (*Result, error) {
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
	result, err := parseReader(f)
	if err != nil {
		return nil, fmt.Errorf("解析日志文件 %q 失败: %w", path, err)
	}
	result.FilePath, result.FileSize = path, info.Size()
	return result, nil
}

type eventBuilder struct {
	event         DiagnosticEvent
	raw           strings.Builder
	message       []string
	authoritative bool
	stackLevelSet bool
}

func newEvent(line int64, header recordHeader, known bool) *eventBuilder {
	if !known {
		header.format, header.level = FormatUnknown, LevelUnknown
	}
	return &eventBuilder{
		event: DiagnosticEvent{
			Format: header.format, Level: header.level,
			StartLine: line, EndLine: line, Timestamp: header.timestamp, Source: header.source,
			Stack: []string{}, BoundaryUncertain: !known,
		},
		authoritative: header.authoritative,
	}
}

func (b *eventBuilder) append(raw, text string, line int64, stack bool) {
	b.raw.WriteString(raw)
	b.event.EndLine = line
	if stack {
		b.event.Stack = append(b.event.Stack, text)
		// Only the first logger frame describes this record's emitting call.
		// A nested caller named Log.Error must not override Log.Message.
		if !b.authoritative && !b.stackLevelSet {
			if level, ok := levelFromStack(text); ok {
				b.event.Level, b.event.Format = level, FormatLegacy
				b.stackLevelSet = true
			}
		}
		return
	}
	// Known record metadata and separators after a stack remain in RawText,
	// but are not part of the error body used for deduplication.
	if isRecordMetadata(text) || (len(b.event.Stack) > 0 && strings.TrimSpace(text) == "") {
		return
	}
	b.message = append(b.message, text)
}

func (b *eventBuilder) finish() DiagnosticEvent {
	b.event.RawText = b.raw.String()
	b.event.Message = strings.Join(b.message, "\n")
	if len(b.message) > 0 {
		b.event.Title = b.message[0]
	}
	return b.event
}

func parseReader(input io.Reader) (*Result, error) {
	result := &Result{Format: FormatUnknown, Events: []DiagnosticEvent{}, Diagnostics: []DiagnosticGroup{}}
	reader := bufio.NewReader(input)
	var current *eventBuilder
	flush := func() {
		if current != nil {
			result.Events = append(result.Events, current.finish())
			current = nil
		}
	}
	for {
		raw, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		if len(raw) > 0 {
			result.TotalLines++
			text := strings.TrimSuffix(strings.TrimSuffix(raw, "\n"), "\r")
			// Strip a UTF-8 BOM for recognition only; RawText remains unchanged.
			if result.TotalLines == 1 {
				text = strings.TrimPrefix(text, "\uFEFF")
			}
			stack := isStackLine(text)
			header, known := recordHeader{}, false
			if !stack {
				header, known = detectHeader(text)
			}
			if known {
				flush()
				current = newEvent(result.TotalLines, header, true)
				// A header's message is body text even if it resembles metadata;
				// only standalone continuation lines may be record metadata.
				current.raw.WriteString(raw)
				current.message = append(current.message, header.message)
			} else {
				// After a stack, a non-frame unmarked message starts a new
				// Unknown record. Its following stack must not attach backwards.
				if current != nil && len(current.event.Stack) > 0 && !stack && strings.TrimSpace(text) != "" && !isRecordMetadata(text) {
					flush()
				}
				if current == nil {
					current = newEvent(result.TotalLines, recordHeader{}, false)
				} else if !stack && strings.TrimSpace(text) != "" && !isRecordMetadata(text) {
					// An unmarked continuation may also be an unmarked new record.
					current.event.BoundaryUncertain = true
				}
				current.append(raw, text, result.TotalLines, stack)
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	flush()
	result.Format = overallFormat(result.Events)
	result.Diagnostics = Deduplicate(result.Events)
	for _, group := range result.Diagnostics {
		result.DiagnosticCount += group.Count
	}
	return result, nil
}

// Unknown preamble does not hide a recognized format. Mixed recognized formats
// produce Unknown at file level; each record still retains its own format.
func overallFormat(events []DiagnosticEvent) Format {
	format := FormatUnknown
	for _, event := range events {
		if event.Format == FormatUnknown {
			continue
		}
		if format != FormatUnknown && format != event.Format {
			return FormatUnknown
		}
		format = event.Format
	}
	return format
}

package logparse

import (
	"regexp"
	"strings"
)

// The unbracketed date/time + level + [category] prefix is observed in public
// Waylith logs. ISO T, fractional seconds and timezone suffixes are accepted too.
var waylithHeader = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:\d{2})?)\s+(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|EXCEPTION|FATAL)\s+\[([^\]\r\n]+)\](?:[ \t]+(.*))?$`)
var legacyLevelHeader = regexp.MustCompile(`(?i)^(?:\[(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|EXCEPTION|FATAL)\][ \t]*|(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|EXCEPTION|FATAL):[ \t]*)(.*)$`)
var legacyErrorHeader = regexp.MustCompile(`(?i)^(?:Error\s+|Exception\s+|Warning\s+|Root level exception\b)`)
var exceptionHeader = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_+` + "`" + `]*\.)*[A-Za-z_][A-Za-z0-9_+` + "`" + `]*Exception(?:[ \t]*:|$)`)

// A method frame must have no space before its type/method separator. Check
// frames before headers, so Log.Error methods cannot create their own events.
var unityFrame = regexp.MustCompile(`^[^\s:]+:[^\s(]+[ \t]*\(`)
var unityLogLevel = regexp.MustCompile(`(?:^|[.:\s])Verse\.Log[.:](Message|Warning|Error|ErrorOnce)(?:_Patch[0-9]+)?[ \t]*\(`)
var referenceMetadata = regexp.MustCompile(`^\[Ref [A-Za-z0-9]+\](?: Duplicate stacktrace, see ref for original)?$`)
var filenameMetadata = regexp.MustCompile(`^\(Filename: .+ Line: -?[0-9]+\)$`)

type recordHeader struct {
	format        Format
	level         Level
	message       string
	timestamp     string
	source        string
	authoritative bool
}

func detectHeader(line string) (recordHeader, bool) {
	if match := waylithHeader.FindStringSubmatch(line); match != nil {
		return recordHeader{FormatWaylith, parseLevel(match[2]), match[4], match[1], match[3], true}, true
	}
	// Indented lines may be inner exceptions or wrapped messages, not records.
	if strings.TrimLeft(line, " \t") != line {
		return recordHeader{}, false
	}
	if match := legacyLevelHeader.FindStringSubmatch(line); match != nil {
		level := match[1]
		if level == "" {
			level = match[2]
		}
		return recordHeader{format: FormatLegacy, level: parseLevel(level), message: match[3], authoritative: true}, true
	}
	if legacyErrorHeader.MatchString(line) || exceptionHeader.MatchString(line) {
		level := LevelError
		if strings.HasPrefix(strings.ToLower(line), "warning") {
			level = LevelWarning
		}
		return recordHeader{format: FormatLegacy, level: level, message: line}, true
	}
	return recordHeader{}, false
}

func parseLevel(text string) Level {
	switch strings.ToUpper(text) {
	case "WARN", "WARNING":
		return LevelWarning
	case "ERROR", "EXCEPTION":
		return LevelError
	default:
		return Level(strings.ToUpper(text))
	}
}

func isStackLine(line string) bool {
	text := strings.TrimSpace(line)
	if strings.HasPrefix(text, "at ") || strings.HasPrefix(text, "(wrapper ") || strings.HasPrefix(text, "--- End of ") {
		return true
	}
	return unityFrame.MatchString(text)
}

func isRecordMetadata(line string) bool {
	text := strings.TrimSpace(line)
	return referenceMetadata.MatchString(text) || filenameMetadata.MatchString(text)
}

func levelFromStack(line string) (Level, bool) {
	if match := unityLogLevel.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
		switch match[1] {
		case "Message":
			return LevelInfo, true
		case "Warning":
			return LevelWarning, true
		default:
			return LevelError, true
		}
	}
	return LevelUnknown, false
}

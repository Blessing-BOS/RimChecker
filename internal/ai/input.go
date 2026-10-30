package ai

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"rimchecker/internal/logparse"
)

type inputGroup struct {
	Level             logparse.Level `json:"level"`
	Source            string         `json:"source"`
	Message           string         `json:"message"`
	Stack             string         `json:"stack"`
	Count             int64          `json:"count"`
	FirstStart        int64          `json:"first_start_line"`
	FirstEnd          int64          `json:"first_end_line"`
	LastStart         int64          `json:"last_start_line"`
	LastEnd           int64          `json:"last_end_line"`
	BoundaryUncertain bool           `json:"boundary_uncertain"`
	ContentTruncated  bool           `json:"content_truncated"`
}

type analysisInput struct {
	Format         logparse.Format `json:"format"`
	Errors         int64           `json:"errors"`
	Warnings       int64           `json:"warnings"`
	TotalGroups    int             `json:"total_groups"`
	IncludedGroups int             `json:"included_groups"`
	OmittedGroups  int             `json:"omitted_groups"`
	Truncated      bool            `json:"truncated"`
	Groups         []inputGroup    `json:"groups"`
}

// BuildInput never includes the file path, raw log, INFO records or timestamps.
// MaxInputChars bounds the complete serialized user message in Unicode codepoints.
func BuildInput(result *logparse.Result, c Config, secrets ...string) (string, error) {
	if result == nil {
		return "", errors.New("缺少结构化解析结果")
	}
	if err := c.Validate(); err != nil {
		return "", err
	}
	in := analysisInput{Format: result.Format, TotalGroups: len(result.Diagnostics), Groups: []inputGroup{}}
	for _, g := range result.Diagnostics {
		if g.Event.Level == logparse.LevelWarning {
			in.Warnings += g.Count
		} else {
			in.Errors += g.Count
		}
	}
	for i, g := range result.Diagnostics {
		if i >= c.MaxEvents {
			break
		}
		message := Redact(g.Event.Message, secrets...)
		stack := Redact(strings.Join(g.Event.Stack, "\n"), secrets...)
		source := Redact(g.Event.Source, secrets...)
		m, mt := clip(message, c.MaxInputChars)
		s, st := clip(stack, c.MaxInputChars)
		src, srct := clip(source, 256)
		in.Groups = append(in.Groups, inputGroup{g.Event.Level, src, m, s, g.Count, g.FirstStartLine, g.FirstEndLine, g.LastStartLine, g.LastEndLine, g.Event.BoundaryUncertain, mt || st || srct})
	}
	for {
		in.IncludedGroups = len(in.Groups)
		in.OmittedGroups = in.TotalGroups - in.IncludedGroups
		in.Truncated = in.OmittedGroups > 0
		for _, g := range in.Groups {
			in.Truncated = in.Truncated || g.ContentTruncated
		}
		b, err := json.Marshal(in)
		if err != nil {
			return "", errors.New("构建分析输入失败")
		}
		if utf8.RuneCount(b) <= c.MaxInputChars {
			return string(b), nil
		}
		if len(in.Groups) == 0 {
			return "", errors.New("输入上限不足以容纳统计元数据")
		}
		last := &in.Groups[len(in.Groups)-1]
		if len(last.Message)+len(last.Stack) > 0 {
			last.Message, _ = clip(last.Message, utf8.RuneCountInString(last.Message)/2)
			last.Stack, _ = clip(last.Stack, utf8.RuneCountInString(last.Stack)/2)
			last.ContentTruncated = true
		} else {
			in.Groups = in.Groups[:len(in.Groups)-1]
		}
	}
}

func clip(text string, n int) (string, bool) {
	count := 0
	for index := range text {
		if count == n {
			return strings.Clone(text[:index]), true
		}
		count++
	}
	return text, false
}

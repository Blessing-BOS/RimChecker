// Package desktop coordinates the shared parser and AI client. Only methods on
// Service are bound; file paths are selected through the native dialog only.
package desktop

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	assets "rimchecker/config"
	"rimchecker/internal/ai"
	"rimchecker/internal/logparse"
)

type Options struct {
	Directory  string
	SelectFile func() (string, error)
	HTTPClient *http.Client
}
type Settings struct {
	BaseURL        string `json:"base_url"`
	Model          string `json:"model"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}
type State struct {
	Settings Settings `json:"settings"`
	HasKey   bool     `json:"has_key"`
	Warning  string   `json:"warning"`
}
type Report struct {
	FileName    string                     `json:"file_name"`
	Format      logparse.Format            `json:"format"`
	TotalLines  int64                      `json:"total_lines"`
	Errors      int64                      `json:"errors"`
	Warnings    int64                      `json:"warnings"`
	Diagnostics []logparse.DiagnosticGroup `json:"diagnostics"`
}
type Preview struct {
	Token    string     `json:"token"`
	Endpoint string     `json:"endpoint"`
	Request  ai.Request `json:"request"`
}
type snapshot struct {
	preview Preview
	config  ai.Config
	key     string
}
type Service struct {
	mu                     sync.Mutex
	options                Options
	settings               Settings
	key, selected, warning string
	result                 *logparse.Result
	revision               uint64
	pending                *snapshot
	cancel                 context.CancelFunc
}

func New(options Options) (*Service, error) {
	if options.Directory == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return nil, errors.New("无法定位用户配置目录")
		}
		options.Directory = filepath.Join(dir, "RimChecker")
	}
	if err := os.MkdirAll(options.Directory, 0700); err != nil {
		return nil, errors.New("无法创建用户配置目录")
	}
	s := &Service{options: options, settings: Settings{TimeoutSeconds: ai.DefaultTimeoutSeconds}}
	data, err := os.ReadFile(s.path("settings.json"))
	if err == nil {
		var saved Settings
		if json.Unmarshal(data, &saved) != nil || s.config(saved).Validate() != nil {
			s.warning = "已保存配置无效，请在设置中重新填写。"
		} else {
			s.settings = saved
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		s.warning = "读取配置失败，请检查用户配置目录权限。"
	}
	return s, nil
}
func (s *Service) path(name string) string { return filepath.Join(s.options.Directory, name) }
func (s *Service) config(v Settings) ai.Config {
	return ai.Config{BaseURL: v.BaseURL, Model: v.Model, TimeoutSeconds: v.TimeoutSeconds, MaxEvents: 20, MaxInputChars: 16000}
}
func (s *Service) invalidate() {
	s.revision++
	s.pending = nil
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Service) GetAIConfig() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return State{s.settings, s.key != "", s.warning}
}
func (s *Service) SaveAIConfig(v Settings) error {
	if err := s.config(v).Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.New("配置编码失败")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// The persistence DTO has no API key field, irrespective of bridge input.
	if err := os.WriteFile(s.path("settings.json"), data, 0600); err != nil {
		return errors.New("保存 API 配置失败")
	}
	s.settings = v
	s.warning = ""
	s.invalidate()
	return nil
}
func (s *Service) SetAPIKey(key string) error {
	if strings.ContainsAny(key, "\r\n") {
		return errors.New("API Key 不能包含换行")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key = strings.TrimSpace(key)
	s.invalidate()
	return nil
}
func (s *Service) GetPrompt() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prompt()
}
func (s *Service) prompt() (string, error) {
	if _, err := os.Stat(s.path("prompt.md")); errors.Is(err, os.ErrNotExist) {
		return assets.DefaultPrompt, nil
	}
	return ai.ReadPrompt(s.path("prompt.md"))
}
func (s *Service) SavePrompt(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ai.SavePrompt(s.path("prompt.md"), text); err != nil {
		return err
	}
	s.invalidate()
	return nil
}

// Returning defaults edits the draft only. Persistence requires SavePrompt.
func (s *Service) ResetPrompt() string { return assets.DefaultPrompt }

func (s *Service) SelectLogFile() (string, error) {
	if s.options.SelectFile == nil {
		return "", errors.New("日志选择器不可用")
	}
	path, err := s.options.SelectFile()
	if err != nil {
		return "", errors.New("选择日志失败")
	}
	if path == "" {
		return "", nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selected = path
	s.result = nil
	s.invalidate()
	return filepath.Base(path), nil
}
func (s *Service) ParseLog() (*Report, error) {
	s.mu.Lock()
	path := s.selected
	s.result = nil
	s.invalidate()
	revision := s.revision
	s.mu.Unlock()
	if path == "" {
		return nil, errors.New("请先选择日志文件")
	}
	result, err := logparse.Parse(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	if revision != s.revision {
		return nil, errors.New("日志或配置已变化，请重新解析")
	}
	if err != nil {
		return nil, errors.New("日志解析失败：文件不存在、不可读或读取失败")
	}
	s.result = result
	report := &Report{FileName: filepath.Base(path), Format: result.Format, TotalLines: result.TotalLines, Diagnostics: result.Diagnostics}
	for _, g := range result.Diagnostics {
		if g.Event.Level == logparse.LevelWarning {
			report.Warnings += g.Count
		} else {
			report.Errors += g.Count
		}
	}
	return report, nil
}
func (s *Service) PreviewAIRequest() (*Preview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = nil
	if s.cancel != nil {
		return nil, errors.New("请先取消或等待当前 AI 请求")
	}
	if s.result == nil {
		return nil, errors.New("请先选择并解析日志")
	}
	if len(s.result.Diagnostics) == 0 {
		return nil, errors.New("当前没有可供 AI 分析的异常组")
	}
	c := s.config(s.settings)
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if s.key == "" {
		return nil, errors.New("请在设置中输入 API Key（仅保留在本次运行内存中）")
	}
	prompt, err := s.prompt()
	if err != nil {
		return nil, err
	}
	input, err := ai.BuildInput(s.result, c, s.key)
	if err != nil {
		return nil, err
	}
	req, err := ai.BuildRequest(c, ai.Redact(prompt, s.key), input)
	if err != nil {
		return nil, err
	}
	endpoint, _ := c.Endpoint()
	p := Preview{Token: rand.Text(), Endpoint: endpoint, Request: req}
	s.pending = &snapshot{p, c, s.key}
	// Deep-copy messages: neither callers nor frontends can change stored content.
	p.Request.Messages = append([]ai.Message(nil), req.Messages...)
	return &p, nil
}
func (s *Service) DiscardPreview() { s.mu.Lock(); defer s.mu.Unlock(); s.pending = nil }
func (s *Service) DiagnoseWithAI(token string, confirmed bool) (string, error) {
	s.mu.Lock()
	if !confirmed {
		s.mu.Unlock()
		return "", ai.ErrUploadDenied
	}
	if s.cancel != nil {
		s.mu.Unlock()
		return "", errors.New("AI 请求正在进行中")
	}
	snap := s.pending
	if snap == nil || token == "" || snap.preview.Token != token {
		s.mu.Unlock()
		return "", errors.New("预览已失效，请重新预览并确认")
	}
	s.pending = nil // consume before I/O; a double click cannot pay twice
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()
	defer func() { cancel(); s.mu.Lock(); s.cancel = nil; s.mu.Unlock() }()
	provider, err := ai.NewClient(snap.config, snap.key, s.options.HTTPClient)
	if err != nil {
		return "", errors.New("AI 配置或密钥无效")
	}
	text, err := provider.Diagnose(ctx, snap.preview.Request, true)
	if errors.Is(ctx.Err(), context.Canceled) {
		return "", errors.New("AI 请求已取消")
	}
	if err != nil {
		return "", err
	}
	return text, nil
}
func (s *Service) CancelDiagnosis() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = nil
	if s.cancel != nil {
		s.cancel()
	}
}

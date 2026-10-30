// Package ai prepares bounded diagnostic inputs and calls a user-selected API.
package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const MaxPromptBytes = 64 * 1024

// DefaultTimeoutSeconds applies when no request timeout has been configured.
const DefaultTimeoutSeconds = 240

type Config struct {
	BaseURL        string `json:"base_url"`
	Model          string `json:"model"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	PromptFile     string `json:"prompt_file"`
	MaxEvents      int    `json:"max_events"`
	MaxInputChars  int    `json:"max_input_chars"`
}

func LoadConfig(path string) (Config, error) {
	c := Config{TimeoutSeconds: DefaultTimeoutSeconds, MaxEvents: 20, MaxInputChars: 16000}
	data, err := readLimitedFile(path, 64*1024)
	if err != nil {
		return c, errors.New("无法读取 AI 配置文件或文件过大")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, errors.New("AI 配置 JSON 无效或包含未知字段（不接受 api_key）")
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return c, errors.New("AI 配置必须只有一个 JSON 对象")
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	if strings.TrimSpace(c.PromptFile) == "" {
		return c, errors.New("prompt_file 不能为空")
	}
	if !filepath.IsAbs(c.PromptFile) {
		c.PromptFile = filepath.Join(filepath.Dir(path), c.PromptFile)
	}
	return c, nil
}

func (c Config) Validate() error {
	if _, err := c.Endpoint(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Model) == "" || len(c.Model) > 256 || strings.ContainsAny(c.Model, "\r\n") {
		return errors.New("model 必须是非空模型名称，最多 256 字节")
	}
	if c.TimeoutSeconds < 1 || c.TimeoutSeconds > 600 {
		return errors.New("timeout_seconds 必须为 1–600")
	}
	if c.MaxEvents < 1 || c.MaxEvents > 1000 {
		return errors.New("max_events 必须为 1–1000")
	}
	if c.MaxInputChars < 512 || c.MaxInputChars > 200000 {
		return errors.New("max_input_chars 必须为 512–200000")
	}
	return nil
}

// Endpoint accepts a base path such as /v1 or /custom/v1, never credentials or
// query parameters. Plain HTTP is limited to loopback development services.
func (c Config) Endpoint() (string, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", errors.New("base_url 无效；不允许内嵌凭据、查询参数或片段")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))) {
		return "", errors.New("base_url 必须使用 HTTPS（回环地址允许 HTTP）")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/chat/completions"
	u.RawPath = ""
	return u.String(), nil
}

func readLimitedFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("file too large")
	}
	return b, nil
}

func validatePrompt(text string) error {
	if strings.TrimSpace(text) == "" || len(text) > MaxPromptBytes || !utf8.ValidString(text) {
		return errors.New("提示词必须为非空 UTF-8 文本，且不超过 64 KiB")
	}
	return nil
}

// ReadPrompt and SavePrompt are file-based boundaries for a future GUI editor.
func ReadPrompt(path string) (string, error) {
	b, err := readLimitedFile(path, MaxPromptBytes)
	if err != nil {
		return "", errors.New("无法读取提示词文件或文件过大")
	}
	text := strings.TrimPrefix(string(b), "\uFEFF")
	if err := validatePrompt(text); err != nil {
		return "", err
	}
	return text, nil
}

// SavePrompt writes only the explicitly selected file; it does not create dirs.
func SavePrompt(path, text string) error {
	if err := validatePrompt(text); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		return errors.New("保存提示词文件失败")
	}
	return nil
}

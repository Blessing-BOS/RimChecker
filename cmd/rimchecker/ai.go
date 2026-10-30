package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"rimchecker/internal/ai"
	"rimchecker/internal/logparse"
)

func runAI(path, configPath, format string, preview, allowUpload bool, stdout, stderr io.Writer) int {
	if !preview && !allowUpload {
		fmt.Fprintln(stderr, ai.ErrUploadDenied)
		return 2
	}
	key := os.Getenv("RIMCHECKER_API_KEY")
	fail := func(err error) int { fmt.Fprintln(stderr, "AI 分析失败："+ai.Redact(err.Error(), key)); return 1 }
	config, err := ai.LoadConfig(configPath)
	if err != nil {
		return fail(err)
	}
	prompt, err := ai.ReadPrompt(config.PromptFile)
	if err != nil {
		return fail(err)
	}
	// Known environment credentials are never included even if a prompt echoes one.
	prompt = ai.Redact(prompt, key)
	result, err := logparse.Parse(path)
	if err != nil {
		return fail(err)
	}
	input, err := ai.BuildInput(result, config, key)
	if err != nil {
		return fail(err)
	}
	request, err := ai.BuildRequest(config, prompt, input)
	if err != nil {
		return fail(err)
	}
	if preview {
		endpoint, _ := config.Endpoint()
		return writeJSON(struct {
			Endpoint string     `json:"endpoint"`
			Notice   string     `json:"notice"`
			Request  ai.Request `json:"request"`
		}{ai.Redact(endpoint, key), "仅预览，未发送。自动清理不保证覆盖全部隐私，请检查全部消息、服务地址与模型后再授权。", request}, stdout, stderr)
	}
	provider, err := ai.NewClient(config, key, nil)
	if err != nil {
		return fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	text, err := provider.Diagnose(ctx, request, allowUpload)
	if err != nil {
		return fail(err)
	}
	if format == "json" {
		return writeJSON(struct {
			Diagnosis string `json:"diagnosis"`
		}{text}, stdout, stderr)
	}
	if _, err := fmt.Fprintln(stdout, text); err != nil {
		fmt.Fprintln(stderr, "输出诊断文本失败")
		return 1
	}
	return 0
}

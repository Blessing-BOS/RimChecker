package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"rimchecker/internal/logparse"
	"rimchecker/internal/logscan"
)

const previewLimit = 10

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("rimchecker", flag.ContinueOnError)
	flags.SetOutput(stderr)
	logPath := flags.String("log", "", "需要扫描的日志文件路径（含空格时请加引号）")
	mode := flags.String("mode", "scan", "处理模式：scan、parse 或 ai")
	format := flags.String("format", "text", "输出格式：text 或 json")
	configPath := flags.String("config", "./config/ai.example.json", "AI JSON 配置路径")
	previewOnly := flags.Bool("preview", false, "仅预览 AI 发送内容，绝不发起网络请求")
	allowUpload := flags.Bool("allow-upload", false, "明确授权本次 AI 请求上传")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "RimChecker - 日志扫描与结构化解析")
		fmt.Fprintln(stderr, "用法：rimchecker -log <日志路径> [-mode scan|parse|ai] [-format text|json]")
		fmt.Fprintln(stderr, "示例：go run ./cmd/rimchecker -log \"./testdata/fixtures/sample.log\"")
		fmt.Fprintln(stderr, "选项：")
		flags.PrintDefaults()
		fmt.Fprintln(stderr, "  -help\n    显示帮助信息")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *logPath == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "参数错误：请通过 -log 指定一个日志路径，不接受额外的位置参数。")
		flags.Usage()
		return 2
	}
	if (*mode != "scan" && *mode != "parse" && *mode != "ai") || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "参数错误：-mode 仅支持 scan、parse 或 ai；-format 仅支持 text 或 json。")
		return 2
	}
	if *mode == "ai" {
		return runAI(*logPath, *configPath, *format, *previewOnly, *allowUpload, stdout, stderr)
	}
	if *previewOnly || *allowUpload {
		fmt.Fprintln(stderr, "参数错误：-preview 和 -allow-upload 仅用于 -mode ai。")
		return 2
	}
	if *mode == "parse" {
		return runParse(*logPath, *format, stdout, stderr)
	}

	result, err := logscan.Scan(*logPath)
	if err != nil {
		fmt.Fprintf(stderr, "扫描失败：%v\n", err)
		return 1
	}
	if *format == "json" {
		return writeJSON(result, stdout, stderr)
	}

	fmt.Fprintln(stdout, "RimChecker - 日志扫描结果")
	fmt.Fprintf(stdout, "\n文件：%s\n路径：%s\n文件大小：%d 字节\n总行数：%d\n",
		filepath.Base(result.FilePath), result.FilePath, result.FileSize, result.TotalLines)
	fmt.Fprintf(stdout, "\n候选异常行数：%d\n  Exception：%d\n  Error：%d\n  Warning：%d\n",
		result.CandidateCount, result.Counts.Exception, result.Counts.Error, result.Counts.Warning)
	fmt.Fprintln(stdout, "\n候选异常详情（最多显示前 10 条）：")
	fmt.Fprintln(stdout, "  行号 | 类型 | 日志摘要")
	preview := result.Candidates
	if len(preview) > previewLimit {
		preview = preview[:previewLimit]
	}
	for _, candidate := range preview {
		fmt.Fprintf(stdout, "  %d | %s | %s\n", candidate.LineNumber, candidate.Type, candidate.Summary)
	}
	if len(preview) == 0 {
		fmt.Fprintln(stdout, "  未发现包含指定关键字的候选异常行。")
	}
	if result.CandidateCount > int64(len(preview)) {
		fmt.Fprintf(stdout, "  另有 %d 条候选异常行未展示。\n", result.CandidateCount-int64(len(preview)))
	}
	fmt.Fprintln(stdout, "\n说明：仅按关键字初步扫描，不能确认真实异常、具体 Mod 或错误根因。")
	return 0
}

func writeJSON(value any, stdout, stderr io.Writer) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(stderr, "JSON 输出失败：%v\n", err)
		return 1
	}
	return 0
}

func runParse(path, format string, stdout, stderr io.Writer) int {
	result, err := logparse.Parse(path)
	if err != nil {
		fmt.Fprintf(stderr, "解析失败：%v\n", err)
		return 1
	}
	if format == "json" {
		return writeJSON(result, stdout, stderr)
	}
	fmt.Fprintln(stdout, "RimChecker - 结构化日志解析结果")
	fmt.Fprintf(stdout, "\n文件：%s\n文件大小：%d 字节\n总行数：%d\n格式：%s\n",
		filepath.Base(result.FilePath), result.FileSize, result.TotalLines, result.Format)
	fmt.Fprintf(stdout, "日志事件数（含 INFO/Unknown）：%d\n错误/警告事件数：%d\n去重后诊断组数：%d\n",
		len(result.Events), result.DiagnosticCount, len(result.Diagnostics))
	uncertain := 0
	for _, event := range result.Events {
		if event.BoundaryUncertain {
			uncertain++
		}
	}
	fmt.Fprintf(stdout, "边界不确定的事件数：%d（完整原文见 JSON 的 events）\n", uncertain)
	fmt.Fprintln(stdout, "\n诊断详情（最多显示前 10 组）：")
	for i, group := range result.Diagnostics {
		if i == previewLimit {
			fmt.Fprintf(stdout, "另有 %d 组未展示，可使用 -format json 查看全部。\n", len(result.Diagnostics)-previewLimit)
			break
		}
		fmt.Fprintf(stdout, "\n[%s] %s\n发生次数：%d\n首次出现：第 %d–%d 行\n末次出现：第 %d–%d 行\n",
			group.Event.Level, group.Event.Title, group.Count,
			group.FirstStartLine, group.FirstEndLine, group.LastStartLine, group.LastEndLine)
		if group.Event.Timestamp != "" || group.Event.Source != "" {
			fmt.Fprintf(stdout, "代表记录时间戳：%s\n来源：%s\n", group.Event.Timestamp, group.Event.Source)
		}
		if group.Event.BoundaryUncertain {
			fmt.Fprintln(stdout, "边界：不确定，保留了未标记的上下文。")
		}
		fmt.Fprintf(stdout, "完整消息：\n%s\n代表堆栈：\n", group.Event.Message)
		for _, frame := range group.Event.Stack {
			fmt.Fprintln(stdout, frame)
		}
		if len(group.Event.Stack) == 0 {
			fmt.Fprintln(stdout, "（未提取到堆栈）")
		}
	}
	if len(result.Diagnostics) == 0 {
		fmt.Fprintln(stdout, "未按当前规则识别到错误或警告；未识别内容仍保留在 JSON 的 events 中。")
	}
	fmt.Fprintln(stdout, "\n说明：仅覆盖有限日志结构，不能据此判断 Mod 根因；Unknown 和边界不确定记录需人工复核。")
	return 0
}

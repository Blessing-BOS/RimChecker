# CLI 使用说明 / CLI Reference

CLI 与桌面版共享扫描、解析和 AI 模块。默认模式是关键词候选扫描，并非准确异常计数。The CLI shares the desktop app's core modules; the default mode is keyword candidate scanning.

## 本地扫描与解析 / Local scanning and parsing

~~~powershell
go run ./cmd/rimchecker -help
go run ./cmd/rimchecker -log "./testdata/fixtures/sample.log"
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode parse
go run ./cmd/rimchecker -log "./testdata/fixtures/waylith-synthetic.log" -mode parse -format json
go build ./cmd/rimchecker
.\rimchecker.exe -log "./testdata/fixtures/sample.log"
~~~

- 默认 scan / text，候选关键词忽略大小写，一行只分类一次，优先级 Exception > Error > Warning。文本显示前 10 条候选。
- parse 提取事件、堆栈并去重，文本显示前 10 组；JSON 的 events 保留全部事件和原文，diagnostics 包含诊断组，故 JSON 可能包含隐私，分享前请检查。
- 支持带引号的空格路径。成功/帮助退出 0，参数错误退出 2，扫描或网络错误退出 1。JSON 成功输出不混入普通提示，错误写入 stderr。
- All fixtures are synthetic. scan reports keyword candidates, not confirmed errors. parse JSON retains raw events and may contain private information. Quoted paths support spaces; exit codes are 0 for success/help, 2 for argument errors, and 1 for file/network errors. Errors go to stderr.

## AI 预览与授权 / AI preview and authorization

~~~powershell
Copy-Item ./config/ai.example.json ./config/ai.local.json
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode ai -config "./config/ai.local.json" -preview
~~~

编辑本地配置的 base_url 和 model。prompt_file 相对于配置所在目录解析，示例指向 config/prompts/default.md。可以指定其他 UTF-8 Markdown；提示词最多 64 KiB，不能为空。

Edit base_url and model. prompt_file is resolved relative to the configuration file; the example points to config/prompts/default.md. Custom prompts must be nonempty UTF-8 and at most 64 KiB.

| 字段 / Field | 说明 / Meaning |
| --- | --- |
| base_url | 必填 API 根地址；追加 /chat/completions / required API root |
| model | 必填模型标识 / required model identifier |
| timeout_seconds | 默认 240，范围 1–600 / default 240, range 1–600 |
| prompt_file | 必填提示词文件 / required prompt file |
| max_events | 默认 20，范围 1–1000 个去重组 / default 20, range 1–1000 groups |
| max_input_chars | 默认 16000，范围 512–200000 个 Unicode 码点 / default 16000, range 512–200000 code points |

字符限制针对日志输入 JSON，不包含系统提示词，不是 Token 限制。配置不接受未知字段或 api_key，API Key 只从 RIMCHECKER_API_KEY 环境变量读取。外部服务 HTTPS、回环地址可 HTTP，不跟随重定向。

The character bound applies to log-input JSON, excludes system prompts, and is not a token budget. Unknown fields and api_key are rejected. The credential comes only from RIMCHECKER_API_KEY. Remote endpoints require HTTPS; loopback HTTP is allowed. Redirects are not followed.

使用隐藏输入，避免将密钥字面量写入终端历史 / Read the key without a literal credential in shell history:

~~~powershell
$credential = Read-Host "API Key" -AsSecureString
$env:RIMCHECKER_API_KEY = [System.Net.NetworkCredential]::new('', $credential).Password
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode ai -config "./config/ai.local.json" -preview
# 检查预览后才执行 / Run only after reviewing the preview:
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode ai -config "./config/ai.local.json" -allow-upload
Remove-Item Env:RIMCHECKER_API_KEY
~~~

-preview 从不发送，且优先于 -allow-upload。缺少这两个选项时直接拒绝。Ctrl+C 取消请求，不自动重试；AI JSON 输出为 diagnosis 字段。

CLI 预览和发送是两个独立进程，不共享桌面版的一次性快照。两次执行间日志、配置或提示词变化会导致输入不同；请使用稳定副本并检查配置。只有桌面版保证发送已确认的快照。

-preview never sends, even when -allow-upload is also present. Without either flag, AI mode refuses to proceed. Ctrl+C cancels a request; there are no automatic retries. AI JSON output has a diagnosis field. CLI preview and upload are separate processes, so changes between them affect the request. Use stable copies. Only the desktop workflow sends the saved, confirmed snapshot.

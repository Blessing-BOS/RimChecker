# RimChecker 架构说明

## 当前模块划分

| 路径 | 职责 |
| --- | --- |
| `cmd/rimchecker/main.go` | 解析参数、处理退出码、输出中文统计和前 10 条候选详情 |
| `internal/logscan/scan.go` | 打开和校验普通文件，逐行扫描，产生独立结果结构体 |
| `internal/logscan/scan_test.go` | 使用临时合成文件验证读取、分类、边界和错误传播 |
| `internal/logparse/types.go` | 结构化事件、去重组和结果结构；JSON 字段定义 |
| `internal/logparse/format.go` | 有限格式记录头、堆栈识别和 Verse 等级证据 |
| `internal/logparse/parse.go` | 流式读行，管理事件边界，保留原文和完整消息 |
| `internal/logparse/dedup.go` | 按消息与堆栈比较，保留首末位置和完整代表事件 |
| `internal/logparse/*_test.go` | 合成格式、边界、去重和原文完整性测试 |
| `cmd/rimchecker/main_test.go` | 验证帮助、参数错误、扫描失败、空格路径和展示限制 |
| `cmd/rimchecker/parse_test.go` | JSON 纯净输出、解析结果和默认模式兼容测试 |
| `testdata/fixtures/sample.log` | 可直接运行的人工合成日志，非真实 RimWorld 数据 |
| `testdata/fixtures/*-synthetic.log` | Legacy / Waylith 合成结构回归样本 |
| `docs/requirements.md` | 范围、暂不实现功能和验收条件 |
| `docs/log-formats.md` | 记录边界、格式依据、去重规则和未覆盖情况 |

Go Module 为 `rimchecker`。源码仓库为 [Blessing-BOS/RimChecker](https://github.com/Blessing-BOS/RimChecker)；当前保留现有 module 与内部导入路径。

## 数据流

默认 scan 模式沿用第一阶段数据流与 API：

```text
用户明确指定 -log 路径
        ↓
CLI 校验参数
        ↓
logscan.Scan(path)
  打开文件 → Stat 校验普通文件并记录大小
        ↓
bufio.Reader.ReadString('\n') 逐行读取
        ↓
总行数累计 → 去除行末 LF/CRLF → 忽略大小写的子串匹配
        ↓
每行最多一种类型 → 分类计数 → 行号、类型、截断摘要
        ↓
Result → CLI 输出统计及前 10 条候选详情
```

`Scan` 不打印内容，不解析命令行，不进行网络请求。错误包装保留底层错误，调用者可使用 `errors.Is` 判断不存在或权限错误。失败时返回错误和空结果，避免把不完整扫描当作成功报告。

## 结果与扫描语义

- `Result`：文件路径、打开时大小（字节）、总行数、候选异常行数、`TypeCounts` 和全部 `CandidateLine` 列表；字段带 JSON 标签，当前没有新增导出接口。
- `CandidateLine`：从 1 开始的行号、`CandidateType` 和摘要。摘要限制为 200 个 Unicode 码点，超长时追加省略号，制表符换为空格。
- 一行命中多个关键字时采用 `Exception > Error > Warning` 优先级，每行只计一次。不进行严重性推断，也不将关键字与具体 Mod 关联。
- `ReadString` 避免 `bufio.Scanner` 默认 token 上限；内存需求约为最长行长度及候选摘要列表之和，不是常量内存。摘要复制后保存，避免短摘要间接保留整条长行。
- 空文件为 0 行；换行分隔的空行照常计数；最后一条非空未终止行计数；结尾换行符不产生额外空行。
- 文件大小是快照，内容若在读取期间变化，本阶段不提供一致性保证。非 UTF-8 文本也不进行识别或转码。

这些都是工具当前的扫描约定；尚未确定的 RimWorld 特定日志格式，必须基于经授权的实际样本再验证。

## Sprint 02 解析流程

```text
CLI -mode parse
    → logparse.Parse(path)：打开并验证普通文件
    → ReadString('\n')：逐行读取，无 64 KiB token 上限
    → 先判断堆栈，再识别明确记录头
    → eventBuilder：组装消息/堆栈/RawText，维护起止行号
    → Events：保留所有等级及 Unknown 的完整记录
    → Deduplicate：筛选诊断等级并按完整内容分组
    → Result：元数据、Events、诊断总次数、Diagnostics
    → CLI 文本报告或纯 JSON 标准输出
```

`logscan` 与 `logparse` 独立，并列使用原始文件作为输入。解析器不依赖被截断的扫描摘要，不让记录边界判断受到逐行关键字计数影响。两者都只依赖标准库，没有网络、界面或参数处理逻辑。

`DiagnosticEvent` 的 `RawText` 保留原始行末和 BOM；语义字段使用统一 LF。`Title` 是第一条消息，`Message` 包含非堆栈正文，`Stack` 保留完整已识别堆栈帧。时间戳和来源只有在记录头明确提供时填写。`BoundaryUncertain` 提示无标记上下文的边界歧义，不表示已检测到某个根因。

文件级 `Format` 汇总识别到的格式，Unknown 前导内容不掩盖已识别格式；若同时存在 Legacy 和 Waylith 则汇总为 Unknown，每条事件仍保留自己的 Format。只命中通用文本规则不能确认游戏版本。

去重使用 Go map，键由等级、来源、完整消息和堆栈文本构成，按首次出现的顺序输出组。堆栈只规范每帧两端空白，不删除方法、偏移、路径或数字；时间戳、行号和原始记录头不进入键。已识别的引用/文件元数据仅保留在 RawText。组保存首次完整事件、次数以及首末起止行号；所有原始出现仍留在 Events。

解析为逐行读取，但结果需要保留所有原文、正文和堆栈，内存随输入规模增长；这与 scan 的截断摘要保留策略不同。读取失败返回 nil 结果及可解包错误，不输出部分成功报告。具体边界与局限见 `log-formats.md`。

## 未来边界

结构化解析与去重已作为独立模块实现。未来分析模块可使用完整事件和诊断组；Mod 关联必须另行建立可验证依据，不能把方法命名或 Source 当成根因。新增格式时可以扩展 `format.go` 的记录头识别与状态规则，无需插件系统或预置空接口。

GUI 在 Sprint 04 使用 Wails v3 + Vue 3 实现。桌面层负责调用业务能力、转换可展示数据和处理用户交互；核心模块保持不依赖桌面框架。

AI 模块在 Sprint 03 实现，网络请求与凭据不进入扫描/解析模块，模型输出与可验证的扫描事实区分。

## Sprint 03 AI 模块

| 文件 | 职责 |
| --- | --- |
| `internal/ai/config.go` | 严格 JSON 配置、URL 与限制校验、ReadPrompt / SavePrompt |
| `internal/ai/redact.go` | 已知密钥替换与常见隐私模式清理 |
| `internal/ai/input.go` | 从去重组构建有效 JSON，清理和限长，标记省略与截断 |
| `internal/ai/client.go` | Provider、纯请求构建、HTTP 实现和安全错误处理 |
| `cmd/rimchecker/ai.go` | 预览、授权、环境密钥读取、取消与终端输出 |
| `config/ai.example.json` | 不含密钥的配置模板 |
| `config/prompts/default.md` | 可编辑的中文诊断提示词 |

调用顺序：CLI 检查预览/授权 → LoadConfig → ReadPrompt → logparse.Parse → BuildInput → BuildRequest → 预览返回或 NewClient + Diagnose。扫描与解析逻辑未被重写。

`Provider` 接口为 `Diagnose(context.Context, Request, bool) (string, error)`；bool 是当次上传许可。HTTP 实现首先拒绝未授权调用，随后使用请求上下文叠加配置超时，执行一次 POST。构造器接受 `*http.Client`，复制后禁用重定向，支持测试/未来上层注入 transport。注入的传输层应可信且不得自带重试或请求日志。

BuildRequest 返回模型与三个 messages：用户编辑的 system 提示词、固定的 system 数据边界说明、日志派生的 user JSON。正文只使用 model/messages，Authorization 单独放在 Bearer 请求头中。协议参考 [OpenAI Chat Completions 文档](https://developers.openai.com/api/reference/resources/chat)。不传 temperature、tools 或 stream 等参数。

BuildInput 不更改原解析结果，不导出文件路径或 RawText；统计包括所有去重组，仅选取限制内的代表正文与堆栈。先对完整字段清理，再按 Unicode 码点裁剪，最后测量整个序列化 JSON 的字符数并保留截断元数据。结果仍是合法 JSON。max_input_chars 不包含 system 提示词，也不是模型 token 限制。

配置相对提示词路径以配置文件目录解析。ReadPrompt / SavePrompt 为未来 Wails 提示词文本框提供读写边界；保存只写显式路径，不创建目录，不持久化密钥。未来 GUI 可持有 Config/提示词文本与 context，先显示 BuildInput/BuildRequest 的预览，再经用户动作调用 Provider；无需依赖 CLI 或改变 logparse。

密钥仅在实际调用的 Client 私有字段与 Header 中使用；CLI 也将其作为已知敏感值用于清理预览/日志派生文本。预览分支不构造 Client，不调用网络。授权不保存，每次运行重新要求。配置、服务地址、模型、提示词由用户控制，日志无法更改这些字段。

成功文本从 choices[0].message.content 读取；非 2xx 不回显错误体。网络错误不包含底层 URL/响应正文；context 错误可通过 errors.Is 判断。响应读取上限 1 MiB，模型 finish_reason=length 视为不完整输出。无自动重试和请求落盘。

边界：清理有漏报/过度清理风险，角色隔离不能保证模型免受提示词注入影响。AI 不执行工具或写入游戏文件。预览/发送是两次独立执行，无法保证两次输入未变化；用户应在日志副本上核对。原解析器仍保留完整日志，输入裁剪不减少解析阶段的内存需求。

## Sprint 04 桌面协调层

CLI 保留上述两次执行行为；GUI 引入内存快照，解决预览/发送不一致问题。

| 路径 | 职责 |
| --- | --- |
| `cmd/rimchecker-desktop/main.go` | 创建 Wails App、原生文件选择器、注册 Service 和主窗口 |
| `internal/desktop/service.go` | 固定用户配置存储、解析协调、统计 DTO、密钥内存、快照与取消 |
| `internal/desktop/service_test.go` | 使用原解析器对照及回环 Mock API 验证关键状态流 |
| `config/assets.go` | 内嵌现有默认提示词，没有复制第二份正文 |
| `frontend/src/App.vue` | 异常列表/详情、设置草稿、预览确认、AI 结果 |
| `frontend/src/style.css` | 深色、可缩放的双栏与弹窗布局 |
| `frontend/assets.go` | 将生产前端资源嵌入 EXE |
| `frontend/bindings` | 官方工具自动生成的调用函数与 DTO 类型 |
| `Taskfile.yml`、`build/windows` | 从官方模板裁剪的 Windows 构建链及 manifest/icon |

Vue 只调用生成的 Service functions，得到 Promise；不直接 fetch 第三方模型，不读写用户任意文件。Go 原生选择器回调返回路径，Service 私有保存路径，ParseLog 无路径参数。Report 的格式、行数、Error/Fatal 与 Warning 次数及 DiagnosticGroup 全部来自 logparse，不从前端关键字推断。列表分页限制 DOM 数量，解析结果仍全部保留在 Go 中。

AI 数据流：已解析结果 → BuildInput(已知内存密钥脱敏) → ReadPrompt / BuildRequest → 私有 snapshot{config,key,request,token} → 返回预览 DTO 的消息副本 → 明确确认 → 验证并消费 token → 现有 ai.Client.Diagnose。发送不重新解析日志或读取配置/提示词文件；磁盘日志在预览后改变也不会改变本次请求。重新选择、解析、保存配置或提示词、更换密钥会递增版本、删除快照并取消活动请求。

Service 使用 mutex 保护状态，但解析与 HTTP 在锁外进行；解析完成需匹配版本才能发布结果。活动 AI 同时最多一个，令牌在网络操作前消费，双击不会产生重复费用。CancelDiagnosis 调用 context cancel，保留 busy 状态直至调用退出。settings.json DTO 根本不含 Key 字段；GetAIConfig 只返回 HasKey，不能读回密钥。用户目录通过 os.UserConfigDir 定位，构造选项仅供 main/测试注入，不绑定给前端。

提示词默认内嵌，ReadPrompt/SavePrompt 只访问固定用户文件。恢复默认返回编辑草稿，保存才写入；设置弹窗的 API 与提示词分别保存，关闭有未保存修改时询问放弃。Markdown 在 html:false 后再经过 DOMPurify 标签白名单，仅用于展示，不能执行 HTML、脚本或加载外部资源。

限制：全量解析仍占用随日志增长的内存，解析任务没有用户取消入口；极大日志的首轮桥接 DTO 也可能较大。没有服务器、聊天历史、自动修复、热更新或桌面热重载。便携 EXE 内嵌资源，需先前端构建再 Go 构建；Taskfile 保证顺序。

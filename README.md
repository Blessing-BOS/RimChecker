[中文](README.md) | [English](README.en.md)

# RimChecker

一款轻量级的 RimWorld（环世界）日志检查与 AI 辅助诊断工具。

源码仓库：[Blessing-BOS/RimChecker](https://github.com/Blessing-BOS/RimChecker)

## 项目介绍

RimChecker 帮助玩家整理日志中的错误、查看调用堆栈并识别重复报错。需要进一步解释时，可以连接自己配置的兼容 AI 服务，获取原因分析和排查建议。

日志解析在本机进行。RimChecker 不管理、排序或修复 Mod，不修改游戏文件或存档，也不提供账号系统或云端服务。AI 输出是需要核实的建议，不是确定的根因结论。

当前准备版本：**v0.9.0-beta**，首个公开测试版本。GitHub Release 尚未发布，项目采用 MIT License。

## 主要功能

- **结构化日志解析**：支持部分 Legacy 与 Waylith 风格记录，提取多行消息、来源、行号及堆栈。
- **异常归类**：查看 Error / Fatal、Warning 的发生次数，并按等级筛选。
- **重复报错合并**：按完整消息和堆栈分组，保留次数、首次和末次出现位置。
- **AI 辅助分析**：解释报错，区分事实与推测，依据日志证据提出可能相关的 Mod 与排查步骤。
- **自定义 AI 配置**：OpenAI-compatible Chat Completions API、模型、超时与可编辑提示词。
- **Windows 桌面界面**：简体中文深色界面、双栏异常列表与详情、可折叠 AI 面板、堆栈及 Markdown 结果复制。
- **命令行**：保留 scan、parse、ai 模式和 JSON 输出。

## 界面预览

仓库尚未收录可公开使用的程序截图，不提供占位图片或虚构截图。截图资源放在 [docs/images](docs/images/README.md)，发布前可补充主界面、异常详情和 AI 预览的真实截图；请先清理路径和个人信息。

## 下载与安装

目标平台为 **Windows x64**，目前在 Windows 11 x64 环境构建和验证，尚未完成干净 Windows 10 环境的兼容性验证。

正式发布后：

1. 打开本仓库的 **Releases** 页面。
2. 下载 RimChecker-v0.9.0-beta-win-x64.zip 及对应的 SHA256 文件。
3. 解压 ZIP，保留随包的第三方许可与使用说明。
4. 运行 RimChecker.exe，无需安装 Go、Node.js 或启动前端开发服务器。

程序依赖 Microsoft Edge WebView2 Runtime，发布包不包含该运行时。若系统缺少它，请从 [Microsoft 官方页面](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) 安装 Evergreen Runtime。

下载链接:https://github.com/Blessing-BOS/RimChecker/releases

## 使用方法

1. 打开 RimChecker，点击「选择日志」。
2. 选择 Player.log、Player-prev.log 或其他日志文件。
3. 点击「开始解析」；选择文件不会自动读取或开始解析。
4. 查看统计，按 Error / Warning 筛选，点击异常查看完整消息、首末行号和堆栈。
5. 按需点击「使用 AI 分析」，检查发送预览后再授权发送。

Windows 常见日志目录：

~~~text
%USERPROFILE%\AppData\LocalLow\Ludeon Studios\RimWorld by Ludeon Studios
~~~

文件选择器在该目录存在时默认进入此处，否则使用系统默认位置；始终允许切换到其他目录。RimWorld 1.7 的日志也可能位于其中的 Logs 子目录，请按实际文件位置选择。

Legacy / Waylith 是本工具识别到的格式风格，不代表准确鉴定游戏版本。建议分析日志副本，避免游戏写入期间内容变化。规则和覆盖范围见 [日志格式说明](docs/log-formats.md)。

## AI 配置

在「设置」中填写：

| 设置 | 说明 |
| --- | --- |
| API Base URL | 服务商提供的 API 根地址，例如 https://api.example.com/v1；程序追加 /chat/completions |
| API Key | 用户自行提供的密钥，仅保留在本次应用运行的内存中 |
| 模型名称 | 服务商支持的模型标识，不绑定特定模型 |
| 超时 | 1–600 秒，默认 240 秒 |
| 诊断提示词 | 可加载、编辑、保存；「恢复默认到草稿」需要再次保存才生效 |

外部服务须使用 HTTPS，仅本机回环地址允许 HTTP；Base URL 不接受用户名、密码、查询参数或片段。

AI 功能需要用户自行提供兼容的 API 服务，服务商可能收取费用。程序不自动重试；用户可取消请求，取消不保证服务商停止计费。

诊断流程：本地解析 → 清理和裁剪输入 → 预览端点、模型、系统提示词及日志输入 → 用户确认 → Go 后端发送一次请求 → 显示 Markdown 结果。

桌面版将非敏感配置保存为 %APPDATA%\RimChecker\settings.json，提示词保存为同目录的 prompt.md。默认提示词内置，安装目录无需可写。API Key 关闭程序后失效，桌面版不读取 CLI 环境密钥。

## 隐私说明

- 本地日志解析不需要联网，程序不自动扫描用户目录。
- 默认不上传日志，AI 输入不包含完整 Player.log、原始日志全文或全部 INFO 记录。
- 只有用户明确确认后，才将预览中的诊断内容发送至自己配置的第三方服务。
- 输入包含去重代表消息、堆栈、次数和行号，经过常见隐私模式过滤和字符数限制；省略或裁剪有明确标记。
- 桌面版最多选取 20 个异常组，日志输入 JSON 最多 16,000 个 Unicode 码点；这不是模型 Token 上限，系统提示词另计。
- GUI 保存一次性请求快照；配置、密钥、提示词或所选日志变化后，旧预览失效。
- API Key 不写入明文配置、localStorage 或默认调试日志。请勿把密钥放入可保存的提示词或 URL。
- 隐私过滤可能漏报或误删信息。发送前请自行检查预览，第三方服务的数据处理规则由服务商决定。
- AI 结果只用于展示，不执行代码或文件操作。Markdown 禁用 HTML，并经过标签白名单清理。

## 从源码构建

技术栈：**Go、Wails v3.0.0、Vue 3、TypeScript、Vite**。在 Windows 项目根目录执行。

已验证工具链：Go 1.27.2、Node.js 26.4.0、npm 11.17.0。Go 最低版本按 go.mod 声明，前端精确依赖由 frontend/package-lock.json 锁定。需安装 WebView2 Runtime，Wails CLI 无需全局安装。

~~~powershell
go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0 doctor
go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0 build
.\bin\RimChecker.exe
~~~

构建顺序：生成 TypeScript bindings → npm ci → 类型检查和前端生产构建 → Windows 资源生成 → Go 生产构建。前端资源和默认提示词嵌入 EXE。首次构建需要下载依赖。

开发检查：

~~~powershell
go test ./...
go vet ./...
go build ./cmd/rimchecker
npm --prefix frontend run typecheck
~~~

完整构建后，生成本地 Beta ZIP 和 SHA256：

~~~powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\package-release.ps1
~~~

脚本收集随包许可、使用说明和 EXE，输出到 dist；不会上传或创建标签。版本、许可检查与发布步骤见 [发布清单](docs/publishing.md)。

目录概览：

~~~text
cmd/rimchecker/           CLI 入口
cmd/rimchecker-desktop/   Wails 桌面入口
internal/logscan/         关键词候选扫描
internal/logparse/        多行解析与去重
internal/ai/              配置、脱敏、输入构建与 API 客户端
internal/desktop/         桌面 Service 协调层
frontend/                Vue 源码、bindings、依赖锁文件
config/                  无密钥示例配置与默认提示词
testdata/fixtures/        合成日志样本
build/windows/           Windows manifest 与图标
scripts/                 本地发布打包
docs/                    架构、格式、截图与发布说明
~~~

CLI 用法：

~~~powershell
go run ./cmd/rimchecker -log "./testdata/fixtures/sample.log"
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode parse -format json
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode ai -config "./config/ai.example.json" -preview
~~~

更多 CLI 参数及授权方式见 [CLI 文档](docs/cli.md)。源码样本和自动化测试数据均为合成数据，不是真实玩家日志。双语 README 的功能、命令和隐私说明应同步维护。

## 已知限制

- 主要面向 Windows x64，桌面界面目前仅有简体中文；其他系统未提供桌面包。
- Legacy / Waylith 兼容性有限，不覆盖所有游戏版本、原生崩溃、嵌套异常或第三方日志包装；不能凭格式标签断定版本。
- 不展开重复堆栈引用，不自动检测或转换编码，预期输入为 UTF-8。
- 结构化解析保留原文及全部事件，超大日志会占用较多内存；解析阶段暂不支持取消。
- AI 解释可能不准确，Mod 关联须有日志证据并由用户验证；信息不足时无法确定根因。
- 兼容 API 的返回格式和模型行为可能不同；不支持流式输出、多轮对话或自动重试。
- Beta 尚未完成干净系统安装验证及所有 DPI / 多显示器组合测试。

## 许可证

RimChecker 采用 [MIT License](LICENSE)。Copyright (c) 2026 Blessing-BOS。

第三方组件继续遵循各自许可证，完整声明见 [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt)，并随 Windows 包分发。RimWorld 名称及相关权利归各自权利人所有；本项目并非官方工具。

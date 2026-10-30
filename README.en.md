[中文](README.md) | [English](README.en.md)

# RimChecker

A lightweight RimWorld log inspector with optional AI-assisted diagnostics.

Source: [Blessing-BOS/RimChecker](https://github.com/Blessing-BOS/RimChecker)

## Overview

RimChecker helps players review game errors, inspect stack traces, and group repeated reports. For further explanation, connect a compatible AI service of your choice to receive possible causes and troubleshooting suggestions.

Logs are parsed locally. RimChecker does not manage, sort, or repair Mods, modify game files or saves, or provide accounts or cloud services. AI output is advice to verify, not a definitive root-cause diagnosis.

Prepared version: **v0.9.0-beta**, the first public beta. The GitHub Release has not been published. The project is licensed under the MIT License.

## Features

- **Structured parsing**: recognizes a subset of Legacy and Waylith records, extracting multiline messages, sources, line numbers, and stacks.
- **Error classification**: displays Error / Fatal and Warning occurrence counts with severity filtering.
- **Duplicate grouping**: compares complete messages and stacks, retaining counts and first/last locations.
- **AI-assisted analysis**: explains errors, separates evidence from speculation, and suggests potentially related Mods and troubleshooting steps when supported by the log.
- **Custom AI configuration**: OpenAI-compatible Chat Completions API, model, timeout, and editable prompts.
- **Windows desktop UI**: a dark Simplified Chinese interface, side-by-side records and details, a collapsible AI panel, and copying of stacks and Markdown results.
- **Command line**: scan, parse, and ai modes, including JSON output.

## Screenshots

No publication-ready screenshots are currently included. There are no mock screenshots or broken placeholder images. See [docs/images](docs/images/README.md) for guidance on adding actual main-window, error-detail, and AI-preview captures. Remove private paths and personal information before publishing.

## Installation

The target platform is **Windows x64**. Builds and verification currently use Windows 11 x64; compatibility on a clean Windows 10 installation has not been verified.

Once the release is published:

1. Open this repository's **Releases** page.
2. Download RimChecker-v0.9.0-beta-win-x64.zip and its SHA256 file.
3. Extract the ZIP, keeping the included third-party notices and instructions.
4. Run RimChecker.exe. Go, Node.js, and a frontend development server are not required.

Microsoft Edge WebView2 Runtime must be installed separately. If it is missing, install the Evergreen Runtime from [Microsoft's official page](https://developer.microsoft.com/en-us/microsoft-edge/webview2/).

Dowload Link:https://github.com/Blessing-BOS/RimChecker/releases

## Usage

1. Open RimChecker and click “选择日志” (Select log).
2. Choose Player.log, Player-prev.log, or another log file.
3. Click “开始解析” (Parse). Selecting a file does not read or parse it automatically.
4. Review the statistics, filter Error / Warning records, and select a record to inspect its complete message, first/last line ranges, and stack.
5. Optionally click “使用 AI 分析” (Analyze with AI), review the outgoing content, and confirm the upload.

Common Windows log directory:

~~~text
%USERPROFILE%\AppData\LocalLow\Ludeon Studios\RimWorld by Ludeon Studios
~~~

The file picker opens here when the directory exists, otherwise it uses the system default. You can always browse elsewhere. RimWorld 1.7 logs may also reside in the Logs subdirectory; choose the actual file location.

Legacy / Waylith labels describe recognized syntax, not an authoritative game-version detection. Prefer a stable copy of the log to avoid changes while the game writes to it. See [log format documentation](docs/log-formats.md) for parsing rules and coverage.

## AI Configuration

Open “设置” (Settings) and enter:

| Setting | Description |
| --- | --- |
| API Base URL | The provider's API root, such as https://api.example.com/v1; RimChecker appends /chat/completions |
| API Key | Your own credential, held in memory for the current application session only |
| Model | A model identifier supported by your provider; no particular model is required |
| Timeout | 1–600 seconds; defaults to 240 |
| Diagnostic prompt | Load, edit, and save the text; restoring defaults updates the draft and requires saving |

Remote endpoints must use HTTPS; HTTP is allowed only for loopback addresses. Embedded credentials, query parameters, and URL fragments are rejected.

You must supply your own compatible API service. Provider charges may apply. RimChecker does not retry automatically. Requests can be canceled, but cancellation does not guarantee the provider stops billing.

Workflow: local parsing → sanitized and bounded input → preview of endpoint, model, system prompts, and log input → explicit confirmation → one request from the Go backend → Markdown results.

Desktop settings are stored in %APPDATA%\RimChecker\settings.json, and your prompt in prompt.md in the same directory. The default prompt is embedded; the installation directory need not be writable. The API Key expires when the application closes. The desktop app does not read the CLI's environment credential.

## Privacy

- Local parsing needs no network access and does not scan user directories automatically.
- Logs are not uploaded by default. AI input does not include the complete Player.log, full raw text, or all INFO records.
- Only after explicit confirmation is the previewed diagnostic content sent to your configured third-party provider.
- Input contains representative messages, stacks, counts, and line numbers from deduplicated groups. Common private-data patterns are filtered, input length is bounded, and omitted or truncated content is marked.
- The desktop app selects at most 20 groups and limits log-input JSON to 16,000 Unicode code points. This is not a model token limit; system prompts are separate.
- The GUI keeps a one-use request snapshot. Changes to settings, credentials, prompts, or the selected log invalidate the old preview.
- API Keys are not saved to plaintext configuration, localStorage, or default debug logs. Do not put credentials in saved prompts or URLs.
- Filtering may miss private data or remove useful evidence. Inspect the preview yourself. Your provider controls how uploaded data is handled.
- AI output is display-only: no code or file operations are executed. Markdown has HTML disabled and is sanitized with an allowed-tag list.

## Building from Source

Stack: **Go, Wails v3.0.0, Vue 3, TypeScript, and Vite**. Run these commands from the project root on Windows.

Verified toolchain: Go 1.27.2, Node.js 26.4.0, npm 11.17.0. The minimum Go version is declared in go.mod; exact frontend dependencies are locked in frontend/package-lock.json. Install WebView2 Runtime. A global Wails CLI installation is not required.

~~~powershell
go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0 doctor
go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0 build
.\bin\RimChecker.exe
~~~

The build generates TypeScript bindings, runs npm ci, type-checks and builds the frontend, generates Windows resources, and builds the production Go executable. Frontend assets and the default prompt are embedded in the EXE. The first build downloads dependencies.

Development checks:

~~~powershell
go test ./...
go vet ./...
go build ./cmd/rimchecker
npm --prefix frontend run typecheck
~~~

After a complete build, create the local beta ZIP and SHA256 file:

~~~powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\package-release.ps1
~~~

The script collects the EXE, notices, and instructions in dist. It does not upload files or create tags. See the [publishing checklist](docs/publishing.md) for versioning, licensing, and release steps.

Project layout:

~~~text
cmd/rimchecker/           CLI entry point
cmd/rimchecker-desktop/   Wails desktop entry point
internal/logscan/         Keyword-based candidate scanning
internal/logparse/        Multiline parsing and deduplication
internal/ai/              Configuration, sanitization, input construction, API client
internal/desktop/         Desktop Service coordination
frontend/                Vue source, bindings, dependency lockfile
config/                  Credential-free example configuration and default prompt
testdata/fixtures/        Synthetic log fixtures
build/windows/           Windows manifest and icon
scripts/                 Local release packaging
docs/                    Architecture, formats, screenshots, release notes
~~~

CLI examples:

~~~powershell
go run ./cmd/rimchecker -log "./testdata/fixtures/sample.log"
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode parse -format json
go run ./cmd/rimchecker -log "./testdata/fixtures/legacy-synthetic.log" -mode ai -config "./config/ai.example.json" -preview
~~~

See [CLI documentation](docs/cli.md) for options and upload authorization. All repository fixtures and automated test data are synthetic, not player logs. Keep features, commands, and privacy statements synchronized between both READMEs.

## Known Limitations

- Primarily targets Windows x64. The desktop interface is currently Simplified Chinese only; desktop packages for other systems are not provided.
- Legacy / Waylith support is partial. Not every game version, native crash, nested exception, or third-party logging wrapper is covered. Format labels do not prove a game version.
- Repeated-stack references are not expanded. Automatic encoding detection or conversion is not supported; UTF-8 input is expected.
- Structured parsing retains raw text and all events, so very large logs consume more memory. Parsing cannot currently be canceled.
- AI explanations may be wrong. Mod associations require log evidence and user verification; incomplete input may not identify a root cause.
- Compatible APIs differ in response formats and model behavior. Streaming, multiround conversations, and automatic retries are not supported.
- Clean-system installation checks and all DPI / multimonitor combinations have not yet been completed for the beta.

## License

RimChecker is licensed under the [MIT License](LICENSE). Copyright (c) 2026 Blessing-BOS.

Third-party components retain their own licenses. Full notices are in [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt) and accompany the Windows package. RimWorld and related rights belong to their respective owners; this is an unofficial tool.

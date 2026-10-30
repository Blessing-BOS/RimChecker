# Beta 发布清单 / Beta Publishing Checklist

## 当前状态 / Current state

- 准备版本 / Prepared version: v0.9.0-beta
- 源码仓库：[Blessing-BOS/RimChecker](https://github.com/Blessing-BOS/RimChecker)，Public；默认分支 main，origin 指向该仓库。版本标签和 GitHub Release 需单独确认后创建。
- Source repository: [Blessing-BOS/RimChecker](https://github.com/Blessing-BOS/RimChecker), public, with main as the default branch and origin pointing to this repository. Tags and GitHub Releases require separate authorization.
- 项目已确定采用 [MIT License](../LICENSE)，版权声明为 Copyright (c) 2026 Blessing-BOS；第三方组件保留各自许可证。
- The project is licensed under the [MIT License](../LICENSE), Copyright (c) 2026 Blessing-BOS. Third-party components retain their own licenses.
- GitHub 源码地址已确定；尚未创建 Release，README 暂无版本下载链接。
- The source repository is established. No Release has been created, so the READMEs do not yet provide version download links.

## 已准备材料 / Prepared materials

| 文件 / File | 用途 / Purpose |
| --- | --- |
| [README.md](../README.md) / [README.en.md](../README.en.md) | 对应的中文 / 英文首页 / Matching Chinese and English home pages |
| [v0.9.0-beta.md](releases/v0.9.0-beta.md) / [English](releases/v0.9.0-beta.en.md) | 首个 Beta 发布说明草稿 / First-beta release note drafts |
| [README.txt](releases/README.txt) | 随 ZIP 分发的双语说明 / Bilingual package instructions |
| [THIRD_PARTY_NOTICES.txt](../THIRD_PARTY_NOTICES.txt) | 随包第三方完整许可与归属 / Full third-party notices and attribution |
| [package-release.ps1](../scripts/package-release.ps1) | 本地白名单打包与校验 / Local whitelist packaging and checksum |
| [docs/images](images/README.md) | 真实截图待补充 / Actual screenshots to be added |
| [构建核对记录](releases/v0.9.0-beta-verification.md) | 构建、包清单、校验值和验证边界 / Build and archive verification |

建议 GitHub 项目描述 / Suggested repository description:

中文：轻量级 RimWorld 日志检查与 AI 辅助诊断工具，支持 Windows 桌面界面和自定义兼容 API。

English: A lightweight RimWorld log inspector with AI-assisted diagnostics, a Windows desktop interface, and support for custom compatible APIs.

## 构建与打包 / Build and package

在 Windows 项目根目录 / From the project root on Windows:

~~~powershell
go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0 build
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\package-release.ps1
~~~

- 输出 dist/RimChecker-v0.9.0-beta-win-x64.zip 及 .zip.sha256；解压内容在 dist 中的同名目录保留。
- Outputs the ZIP and checksum under dist; the staging directory is retained there.
- 已有同名输出时脚本拒绝覆盖。先将旧文件另存，再重新打包；不需要删除源码或用户配置。
- The script refuses to overwrite existing output. Move old artifacts aside before repackaging.
- 脚本需要 Go、Node.js、已安装的 npm 依赖和刚完成的生产 EXE。用 EXE build info 确认 production、trimpath、Windows amd64，并检查依赖版本与本地模块/锁文件一致。
- Requires Go, Node.js, installed npm dependencies, and a fresh production EXE. Build info is checked for production, trimpath, Windows amd64, and matching local dependency versions.
- 发布脚本生成根目录第三方声明，白名单仅收录 EXE、LICENSE、THIRD_PARTY_NOTICES.txt 和 README.txt。缺少项目 LICENSE 时拒绝打包。
- Notices are generated locally. The archive contains only the EXE, LICENSE, THIRD_PARTY_NOTICES.txt, and README.txt. Packaging fails if the project LICENSE is missing.
- 变更依赖或二进制后必须重新生成许可和哈希。新增的组件没有许可文本时应停止打包并核查，不可静默跳过。
- Regenerate notices and checksums after changing dependencies or binaries. Investigate missing license text rather than silently omitting it.
- 第三方清单依据 Windows EXE 实际链接模块及 npm 生产依赖闭包；保守保留部分构建时被移除的编译器/类型包许可。WebView2 Runtime 未打包。
- Notices cover linked Windows Go modules and the production npm dependency closure, conservatively retaining some compiler/type package notices removed by bundling. WebView2 Runtime is not bundled.

## 上传前人工检查 / Manual steps before publication

1. 核对 MIT LICENSE 的年份与版权持有人为 2026 Blessing-BOS，确保两份 README、发布说明及 ZIP 中的许可信息一致；保留全部第三方声明。
   Verify the MIT LICENSE identifies 2026 Blessing-BOS and matches both READMEs, release notes, and the ZIP. Preserve all third-party notices.
2. 使用合成日志补充真实界面截图，检查路径和敏感信息，更新两份 README。
   Add actual application screenshots using synthetic logs, remove private data, and update both READMEs.
3. 在干净 Windows x64 环境验证 ZIP 解压启动与 WebView2 依赖；按需验证 Windows 10 和常见 DPI / 多显示器配置。
   Verify extraction, startup, and WebView2 requirements on clean Windows x64; check Windows 10 and common DPI / multimonitor setups as needed.
4. 核对现有 GitHub 仓库、可见性和描述；创建 Release 后，再为两份 README 添加实际下载链接。
   Verify the existing repository, visibility, and description. Add actual download links to both READMEs after creating the Release.
5. 审核候选提交。保留源码、go.mod/go.sum、前端锁文件、bindings、Windows 构建资源、默认提示词及三个合成 fixture；排除私人配置、真实日志、密钥、.tools、bin、dist、node_modules 与生成的 syso。
   Review candidate files, keeping source, module files, frontend lockfile and bindings, build resources, default prompts, and synthetic fixtures. Exclude private settings, player logs, credentials, temporary files, and build outputs.
6. 检查 ZIP 白名单、SHA256 和 EXE 版本对应关系。当前版本名体现在发布材料与包名中；没有新增应用内版本界面或改写可执行文件版本资源。
   Check the archive whitelist, SHA256, and binary correspondence. The release name appears in package and documentation names; no in-app version UI or executable version resource was added.
7. 正式发布前删除两种 Release Notes 的“发布草稿”语句并更新实际发布状态。只上传选定的 ZIP、校验文件及许可必要附件，不上传开发目录。
   Update both release-note drafts to the actual publication state. Upload only the chosen archive, checksum, and required licensing materials.

后续修改时可在本地审阅提交内容；以下命令不包含远程上传：

Review subsequent staged changes locally. These commands do not upload anything:

~~~powershell
git add .gitattributes .gitignore LICENSE README.md README.en.md THIRD_PARTY_NOTICES.txt go.mod go.sum Taskfile.yml cmd internal frontend config testdata build docs scripts
git diff --cached --check
git diff --cached --stat
git diff --cached
~~~

.gitignore 是防止误提交的辅助规则，不是秘密检测器。合成测试中的虚构密钥用于测试安全行为；不要添加真实凭据。用于安全检查的临时审计元数据只在被忽略的 .tools 中。

.gitignore reduces accidental inclusion but is not a secret scanner. Synthetic credentials in tests verify safety behavior; never add real keys. Temporary audit metadata exists only inside the ignored .tools directory.

## Beta 验证边界 / Verification boundaries

本次不重做已由用户验证的全套 GUI 功能。不使用真实 API Key，不调用付费 API，不修改解析、去重、AI 输入限制或授权逻辑。详细构建结果和启动检查见本版本核对记录。

This preparation does not repeat the user's complete GUI validation. No real API keys or paid requests are used, and parsing, grouping, input bounds, and upload authorization are unchanged. See the version verification record for build and startup checks.

代码签名、安装器和正式应用图标可在后续发布中处理；当前仍使用带归属声明的 Wails 模板图标。不要移除第三方版权和许可文本。

Signing, an installer, and a dedicated application icon can be addressed later. The current template icon retains attribution. Keep required third-party copyright and license text.

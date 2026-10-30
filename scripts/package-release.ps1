# Package only explicit release files. Run the complete Wails build first.
[CmdletBinding()]
param([ValidatePattern('^v[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9.-]+)?$')][string]$Version = 'v0.9.0-beta')
$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
$utf8 = [Text.UTF8Encoding]::new($false)
Push-Location $repo
try {
    $licensePath = Join-Path $repo 'LICENSE'
    if (!(Test-Path -LiteralPath $licensePath -PathType Leaf)) { throw 'Project LICENSE is required for release packaging.' }
    $exe = Join-Path $repo 'bin/RimChecker.exe'
    if (!(Test-Path -LiteralPath $exe -PathType Leaf)) { throw 'Run the Wails production build first.' }
    $buildInfo = (& go version -m $exe) -join [Environment]::NewLine
    if ($LASTEXITCODE -ne 0 -or $buildInfo -notmatch 'GOOS=windows' -or $buildInfo -notmatch 'GOARCH=amd64' -or $buildInfo -notmatch '-trimpath=true' -or $buildInfo -notmatch '-tags=production') {
        throw 'Expected a Windows amd64 Wails production build with trimpath.'
    }
    $name = "RimChecker-$Version-win-x64"
    $stage = Join-Path $repo "dist/$name"
    $zip = Join-Path $repo "dist/$name.zip"
    $checksum = "$zip.sha256"
    foreach ($target in @($stage, $zip, $checksum)) {
        if (Test-Path -LiteralPath $target) { throw "Output already exists; preserve or move it before repackaging: $target" }
    }

    $notices = [Text.StringBuilder]::new()
    [void]$notices.AppendLine("RimChecker $Version - Third-party notices")
    [void]$notices.AppendLine('These licenses apply to the named third-party components, not to RimChecker original code.')
    [void]$notices.AppendLine('RimChecker original code is licensed under the MIT License; see LICENSE. These notices cover third-party components only.')
    [void]$notices.AppendLine('Includes linked Go modules, Wails native components and templates, and the production npm dependency closure.')
    [void]$notices.AppendLine('The npm closure conservatively includes compiler/type packages; some are removed by the production bundler.')
    [void]$notices.AppendLine('WebView2 Runtime is installed separately and is not redistributed in this archive.')
    function Add-Notice([string]$Label, [string]$Text) {
        [void]$notices.AppendLine()
        [void]$notices.AppendLine(('=' * 72))
        [void]$notices.AppendLine($Label)
        [void]$notices.AppendLine(('-' * 72))
        [void]$notices.AppendLine($Text.TrimEnd())
    }
    function Add-LicenseFiles([string]$Label, [string]$Directory) {
        $files = @(Get-ChildItem -LiteralPath $Directory -File | Where-Object { $_.Name -match '^(LICENSE|LICENCE|COPYING|NOTICE|PATENTS)([.-].*)?$' } | Sort-Object Name)
        if ($files.Count -eq 0) { throw "Missing license text: $Label" }
        foreach ($file in $files) { Add-Notice "$Label / $($file.Name)" ([IO.File]::ReadAllText($file.FullName)) }
    }
    $goRoot = (& go env GOROOT).Trim()
    Add-Notice 'Go runtime and standard library / LICENSE' ([IO.File]::ReadAllText((Join-Path $goRoot 'LICENSE')))
    foreach ($vendor in @('crypto', 'net', 'sys', 'text')) {
        Add-LicenseFiles "Go standard-library vendored golang.org/x/$vendor" (Join-Path $goRoot "src/vendor/golang.org/x/$vendor")
    }
    $wailsDirectory = $null
    foreach ($match in [regex]::Matches($buildInfo, '(?m)^\s+dep\s+(\S+)\s+(\S+)')) {
        $moduleName = $match.Groups[1].Value
        $module = (& go list -m -json $moduleName | Out-String | ConvertFrom-Json)
        if ($LASTEXITCODE -ne 0 -or !$module.Dir -or $module.Version -ne $match.Groups[2].Value) { throw "Module version mismatch: $moduleName" }
        Add-LicenseFiles "$moduleName $($module.Version)" $module.Dir
        if ($moduleName -eq 'github.com/wailsapp/wails/v3') { $wailsDirectory = $module.Dir }
    }
    if (!$wailsDirectory) { throw 'Wails not found in executable build info.' }
    Add-LicenseFiles 'Wails internal/go-common-file-dialog' (Join-Path $wailsDirectory 'internal/go-common-file-dialog')
    Add-LicenseFiles 'Wails internal/webview2/webviewloader' (Join-Path $wailsDirectory 'internal/webview2/webviewloader')
    $guidSource = [IO.File]::ReadAllText((Join-Path $wailsDirectory 'internal/webview2/pkg/edge/guid.go'))
    $inlineLicense = [regex]::Match($guidSource, '(?s)/\*(.*?)\*/').Groups[1].Value
    if ($inlineLicense -notmatch 'MIT License') { throw 'Missing inline WebView2 COM license.' }
    Add-Notice 'Wails WebView2 COM/GUID adaptations (also in pkg/webview2/com.go)' $inlineLicense.Trim()
    Add-Notice 'Wails runtime, template manifest, and template icon attribution' 'The Wails runtime, template-derived build configuration, manifest, and icon are from Wails v3.0.0. The Wails MIT license reproduced above applies. The template icon is not an original RimChecker brand asset.'

    # PowerShell 5.1 cannot deserialize package-lock's empty root-package key.
    # Node is already required by the build; emit only named production entries.
    $packages = (& node -e "const lock=require('./frontend/package-lock.json'); process.stdout.write(JSON.stringify(Object.entries(lock.packages).filter(([name,value])=>name&&!value.dev).map(([path,value])=>({path,...value}))))" | ConvertFrom-Json)
    if ($LASTEXITCODE -ne 0) { throw 'Unable to read the npm lockfile.' }
    foreach ($entry in ($packages | Sort-Object path)) {
        $directory = Join-Path (Join-Path $repo 'frontend') $entry.path
        $pkg = Get-Content -LiteralPath (Join-Path $directory 'package.json') -Raw | ConvertFrom-Json
        if ($pkg.version -ne $entry.version) { throw "npm version mismatch: $($entry.path)" }
        $label = "$($pkg.name) $($pkg.version) [$($pkg.license)]"
        if ($pkg.name -eq '@wailsio/runtime') {
            Add-Notice $label 'Covered by the Wails MIT license reproduced above (the npm package does not include a separate license file).'
        } else {
            Add-LicenseFiles $label $directory
        }
        if ($pkg.name -eq 'dompurify') {
            Add-Notice 'DOMPurify copyright and licensing' 'Copyright Cure53 and other contributors. DOMPurify is available under MPL-2.0 OR Apache-2.0; it is redistributed here under Apache-2.0. Both supplied license texts are retained. Source: https://github.com/cure53/DOMPurify'
        }
    }
    $noticePath = Join-Path $repo 'THIRD_PARTY_NOTICES.txt'
    [IO.File]::WriteAllText($noticePath, $notices.ToString(), $utf8)
    New-Item -ItemType Directory -Path $stage | Out-Null
    Copy-Item -LiteralPath $exe -Destination (Join-Path $stage 'RimChecker.exe')
    Copy-Item -LiteralPath $noticePath -Destination (Join-Path $stage 'THIRD_PARTY_NOTICES.txt')
    Copy-Item -LiteralPath (Join-Path $repo 'docs/releases/README.txt') -Destination (Join-Path $stage 'README.txt')
    Copy-Item -LiteralPath $licensePath -Destination (Join-Path $stage 'LICENSE')
    Compress-Archive -LiteralPath (Join-Path $stage 'RimChecker.exe'), (Join-Path $stage 'LICENSE'), (Join-Path $stage 'THIRD_PARTY_NOTICES.txt'), (Join-Path $stage 'README.txt') -DestinationPath $zip -CompressionLevel Optimal
    $hash = (Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
    [IO.File]::WriteAllText($checksum, "$hash  $name.zip" + [Environment]::NewLine, $utf8)
    Write-Output "Archive: $zip"
    Write-Output "SHA256: $hash"
} finally {
    Pop-Location
}

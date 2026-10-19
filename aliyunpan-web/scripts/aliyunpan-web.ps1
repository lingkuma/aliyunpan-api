param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Args
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$skillDir = Split-Path -Parent $scriptDir
$exe = Join-Path $skillDir "bin\aliyunpan-web.exe"

# Prefer the bundled binary so the skill folder can be moved anywhere.
if (Test-Path $exe) {
    & $exe @Args
    exit $LASTEXITCODE
}

# Fallback: locate the aliyunpan-api repo and run from source.
$dir = $env:ALIYUNPAN_REPO
if (-not $dir) {
    $dir = $skillDir
    while ($dir -and -not (Test-Path (Join-Path $dir "go.mod"))) {
        $parent = Split-Path -Parent $dir
        if ($parent -eq $dir) { $dir = $null; break }
        $dir = $parent
    }
}

$pkg = $null
if ($dir) {
    $pkg = Join-Path $dir "aliyunpan-web\cmd\aliyunpan-web"
}
if (-not $dir -or -not (Test-Path $pkg)) {
    Write-Error "Cannot find aliyunpan-api repo. Run scripts\build.ps1 to create bin\aliyunpan-web.exe."
    exit 1
}

Push-Location $dir
try {
    & go run ./aliyunpan-web/cmd/aliyunpan-web @Args
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}

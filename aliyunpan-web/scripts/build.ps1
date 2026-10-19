param(
    [string]$Output = "",
    [string]$Repo = ""
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$skillDir = Split-Path -Parent $scriptDir

$dir = $Repo
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
    Write-Error "Cannot find aliyunpan-api repo. Use -Repo <path> to specify the repo root."
    exit 1
}

$target = $Output
if (-not $target) {
    $target = Join-Path $skillDir "bin\aliyunpan-web.exe"
}
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null

Push-Location $dir
try {
    & go build -o $target ./aliyunpan-web/cmd/aliyunpan-web
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go build failed."
        exit $LASTEXITCODE
    }
    Write-Host "Built: $target"
}
finally {
    Pop-Location
}

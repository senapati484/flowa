param(
    [string]$InstallPath
)

$ErrorActionPreference = "Stop"

if (-not $InstallPath -or $InstallPath.Trim().Length -eq 0) {
    if ($env:ProgramFiles -and ([bool](whoami /groups | Select-String "S-1-5-32-544" -Quiet))) {
        $InstallPath = Join-Path $env:ProgramFiles "Flowa"
    } else {
        $InstallPath = Join-Path $env:LOCALAPPDATA "Programs/Flowa"
    }
}

Write-Host "🚀 Installing Flowa into $InstallPath" -ForegroundColor Cyan

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is not installed. Install it from https://go.dev/dl and retry."
}

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location $repoRoot

Write-Host "Building CLI..."
go build -o flowa.exe ./cmd/flowa

New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null
Copy-Item flowa.exe (Join-Path $InstallPath "flowa.exe") -Force

Write-Host "✓ Flowa installed." -ForegroundColor Green
Write-Host "Add $InstallPath to your PATH if it isn't already."
Write-Host "Run 'flowa --help' to get started!"

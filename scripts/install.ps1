#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"
Write-Host "ProRouter Installer" -ForegroundColor Cyan
Write-Host "===================" -ForegroundColor Cyan

$repo = "FernandoBolzan/ProRouter"
$installDir = if ($env:PROROUTER_INSTALL_DIR) { $env:PROROUTER_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\ProRouter" }
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "x86" }

# Get latest release version
try {
  $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -ErrorAction Stop
  $version = $release.tag_name -replace "^v", ""
} catch {
  Write-Host "No GitHub release found. Attempting 'go install'..." -ForegroundColor Yellow
  try {
    go install "github.com/$repo/gateway-go/cmd/prorouter@latest"
    Write-Host "`nProRouter installed via 'go install'!" -ForegroundColor Green
    Write-Host "Run 'prorouter init' to get started." -ForegroundColor Green
    exit 0
  } catch {
    Write-Host "`nCould not install ProRouter automatically." -ForegroundColor Red
    Write-Host ""
    Write-Host "Option 1: Install Go from https://go.dev/dl/, then run:" -ForegroundColor Yellow
    Write-Host "  go install github.com/FernandoBolzan/ProRouter/gateway-go/cmd/prorouter@latest" -ForegroundColor White
    Write-Host ""
    Write-Host "Option 2: Download a prebuilt binary from:" -ForegroundColor Yellow
    Write-Host "  https://github.com/FernandoBolzan/ProRouter/releases" -ForegroundColor White
    Write-Host ""
    exit 1
  }
}

$filename = "prorouter_${version}_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/v${version}/${filename}"

Write-Host "Downloading ProRouter v${version} (windows/${arch})..."
try {
  Invoke-WebRequest -Uri $url -OutFile "$env:TEMP\$filename" -UseBasicParsing -ErrorAction Stop
} catch {
  Write-Host "Binary download failed: $_" -ForegroundColor Red
  exit 1
}

Write-Host "Extracting..."
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Expand-Archive "$env:TEMP\$filename" -DestinationPath $installDir -Force
Remove-Item "$env:TEMP\$filename" -Force

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
}

Write-Host "`nProRouter v${version} installed to: $installDir" -ForegroundColor Green
Write-Host "Run 'prorouter init' to get started." -ForegroundColor Green

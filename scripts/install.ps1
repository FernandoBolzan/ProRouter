#!/usr/bin/env pwsh
# ProRouter Windows Installer
$ErrorActionPreference = "Stop"
Write-Host "ProRouter Installer" -ForegroundColor Cyan
Write-Host "===================" -ForegroundColor Cyan

$repo = "prorouter/prorouter"
$installDir = if ($env:PROROUTER_INSTALL_DIR) { $env:PROROUTER_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\ProRouter" }

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "x86" }

# Get latest release
$apiUrl = "https://api.github.com/repos/$repo/releases/latest"
try {
  $release = Invoke-RestMethod -Uri $apiUrl
  $version = $release.tag_name -replace "^v", ""
} catch {
  $version = "0.1.0"
}

$filename = "prorouter_${version}_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/v${version}/${filename}"

Write-Host "Downloading ProRouter v${version} (windows/${arch})..."
Invoke-WebRequest -Uri $url -OutFile "$env:TEMP\$filename" -UseBasicParsing

Write-Host "Extracting..."
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Expand-Archive "$env:TEMP\$filename" -DestinationPath $installDir -Force
Remove-Item "$env:TEMP\$filename" -Force

# Add to PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
}

Write-Host "`nProRouter v${version} installed to: $installDir" -ForegroundColor Green
Write-Host "Run 'prorouter init' to get started." -ForegroundColor Green

# Distribution & Installation System

## 1. Multi-Channel Distribution Architecture

### 1.1 Binary Distribution (Primary)

**Platform Support:**
| OS | Arch | Status |
|---|---|---|
| Linux | amd64, arm64, armv7 | ✅ Supported |
| macOS | amd64, arm64 (Apple Silicon) | ✅ Supported |
| Windows | amd64, arm64 | ✅ Supported |
| FreeBSD | amd64 | ⚠️ Community |

**Release Artifacts (per version):**
```
prorouter_<version>_linux_amd64.tar.gz
prorouter_<version>_linux_arm64.tar.gz
prorouter_<version>_darwin_amd64.tar.gz
prorouter_<version>_darwin_arm64.tar.gz
prorouter_<version>_windows_amd64.zip
prorouter_<version>_windows_arm64.zip
prorouter_<version>_checksums.txt        # SHA256 checksums
prorouter_<version>_checksums.txt.sig    # Cosign signature
```

### 1.2 Homebrew (macOS/Linux)

**Tap Repository:** `github.com/FernandoBolzan/homebrew-tap`

```ruby
# Formula: prorouter.rb
class Prorouter < Formula
  desc "Open-source LLM gateway and router"
  homepage "https://github.com/FernandoBolzan/ProRouter"
  version "<version>"
  license "MIT"

  on_macos do
  on_arm { url "https://github.com/FernandoBolzan/ProRouter/releases/download/v<version>/prorouter_<version>_darwin_arm64.tar.gz" }
  on_intel { url "https://github.com/FernandoBolzan/ProRouter/releases/download/v<version>/prorouter_<version>_darwin_amd64.tar.gz" }
  end

  on_linux do
  on_arm { url "https://github.com/FernandoBolzan/ProRouter/releases/download/v<version>/prorouter_<version>_linux_arm64.tar.gz" }
  on_intel { url "https://github.com/FernandoBolzan/ProRouter/releases/download/v<version>/prorouter_<version>_linux_amd64.tar.gz" }
  end

  def install
    bin.install "prorouter"
  end

  service do
    run [opt_bin/"prorouter", "serve"]
    keep_alive true
    log_path var/"log/prorouter.log"
    error_log_path var/"log/prorouter.err"
  end
end
```

### 1.3 Scoop (Windows)

**Bucket Repository:** `github.com/FernandoBolzan/scoop-bucket`

```json
{
  "version": "<version>",
  "description": "Open-source LLM gateway and router",
  "homepage": "https://github.com/FernandoBolzan/ProRouter",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/FernandoBolzan/ProRouter/releases/download/v<version>/prorouter_<version>_windows_amd64.zip",
      "hash": "<sha256>"
    }
  },
  "bin": "prorouter.exe",
  "checkver": {
    "github": "https://github.com/FernandoBolzan/ProRouter"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/FernandoBolzan/ProRouter/releases/download/v$version/prorouter_$version_windows_amd64.zip"
      }
    }
  }
}
```

### 1.4 NPM Package (`@prorouter/cli`)

The NPM package acts as a **binary downloader and wrapper**, not a recompile. It downloads the correct platform binary on `npm install` / `npm i -g @prorouter/cli`.

```json
{
  "name": "@prorouter/cli",
  "version": "<version>",
  "description": "ProRouter CLI - Universal LLM Router",
  "bin": {
    "prorouter": "./bin/run.js"
  },
  "scripts": {
    "postinstall": "node ./scripts/download-binary.js",
    "preuninstall": "node ./scripts/cleanup.js"
  },
  "optionalDependencies": {
    "@prorouter/darwin-arm64": "<version>",
    "@prorouter/darwin-x64": "<version>",
    "@prorouter/linux-arm64": "<version>",
    "@prorouter/linux-x64": "<version>",
    "@prorouter/win32-x64": "<version>"
  }
}
```

**Binary Download Script (`scripts/download-binary.js`):**
```javascript
const { platform, arch } = process;
const map = {
  'darwin-arm64': '@prorouter/darwin-arm64',
  'darwin-x64': '@prorouter/darwin-x64',
  'linux-arm64': '@prorouter/linux-arm64',
  'linux-x64': '@prorouter/linux-x64',
  'win32-x64': '@prorouter/win32-x64',
};

const pkg = map[`${platform}-${arch}`];
if (!pkg) {
  console.error(`Unsupported platform: ${platform}-${arch}`);
  process.exit(1);
}

const binaryPath = require.resolve(`${pkg}/bin/prorouter`);
const fs = require('fs');
fs.chmodSync(binaryPath, 0o755);
```

### 1.5 Docker Images

**Multi-Architecture Images:**
- `prorouter/gateway:latest` - Gateway + Dashboard embedded
- `prorouter/gateway:<version>` - Pinned version
- `prorouter/gateway:<version>-slim` - Gateway only (no dashboard)

```dockerfile
# Dockerfile.gateway (multi-stage build)
FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY gateway-go/ .
RUN go build -o /prorouter ./cmd/prorouter

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /prorouter /usr/local/bin/prorouter
EXPOSE 8080
ENTRYPOINT ["prorouter"]
CMD ["serve"]
```

### 1.6 One-Line Install Script

**Unix (`curl -fsSL https://raw.githubusercontent.com/FernandoBolzan/ProRouter/main/scripts/install.sh | sh`):**
```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="prorouter/prorouter"
INSTALL_DIR="${PROROUTER_INSTALL_DIR:-/usr/local/bin}"
VERSION="${PROROUTER_VERSION:-latest}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
fi

# Download binary
FILENAME="prorouter_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "Downloading ProRouter $VERSION ($OS/$ARCH)..."
curl -fsSL "$URL" -o "/tmp/$FILENAME"

# Extract and install
tar -xzf "/tmp/$FILENAME" -C "/tmp/"
sudo mv "/tmp/prorouter" "$INSTALL_DIR/prorouter"
chmod +x "$INSTALL_DIR/prorouter"
rm "/tmp/$FILENAME"

echo "ProRouter installed successfully!"
echo "Run 'prorouter init' to get started."
```

**Windows (`irm https://raw.githubusercontent.com/FernandoBolzan/ProRouter/main/scripts/install.ps1 | iex`):**
```powershell
# install.ps1
$repo = "prorouter/prorouter"
$installDir = if ($env:PROROUTER_INSTALL_DIR) { $env:PROROUTER_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\ProRouter" }

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "arm64" }

# Get latest release
$apiUrl = "https://api.github.com/repos/$repo/releases/latest"
$release = Invoke-RestMethod -Uri $apiUrl
$version = $release.tag_name

# Download
$filename = "prorouter_${version}_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/$version/$filename"
Write-Host "Downloading ProRouter $version..."
Invoke-WebRequest -Uri $url -OutFile "$env:TEMP\$filename"

# Extract
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Expand-Archive "$env:TEMP\$filename" -DestinationPath $installDir -Force
Remove-Item "$env:TEMP\$filename"

# Add to PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
}

Write-Host "ProRouter installed successfully!"
Write-Host "Run 'prorouter init' to get started."
```

---

## 2. Auto-Update System

### 2.1 Update Channels
| Channel | Frequency | Stability |
|---|---|---|
| `stable` | Monthly releases | High |
| `beta` | Weekly | Medium |
| `nightly` | Daily (autobuild) | Low |

### 2.2 Update Flow
```
User: prorouter update
1. CLI checks current version vs latest in channel
2. Downloads new binary to temp directory
3. Verifies Cosign signature
4. Swaps binary atomically (rename on Unix, move on Windows)
5. Runs migrations if needed
6. Restarts server if running as service
```

### 2.3 Update Configuration (`~/.prorouter/update.json`)
```json
{
  "channel": "stable",
  "last_check": "2026-07-01T12:00:00Z",
  "version": "0.1.0",
  "rollback_versions": ["0.0.9", "0.0.8"]
}
```

---

## 3. Supply Chain Security

| Measure | Implementation |
|---|---|
| **Binary Signing** | Cosign (Sigstore) signatures on all releases |
| **Provenance** | SLSA Level 3 attestations via GitHub Actions |
| **SBOM** | SPDX JSON attached to every release |
| **Docker Signing** | Docker Content Trust (DCT) |
| **NPM Signing** | npm provenance (GitHub OIDC) |
| **Dependency Scanning** | Dependabot + Trivy in CI |

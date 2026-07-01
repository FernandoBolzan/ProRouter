#!/usr/bin/env node
const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const { platform, arch } = process;
const version = require('../package.json').version;

const platformMap = {
  'darwin-arm64': 'darwin_arm64',
  'darwin-x64': 'darwin_amd64',
  'linux-arm64': 'linux_arm64',
  'linux-x64': 'linux_amd64',
  'win32-x64': 'windows_amd64',
  'win32-arm64': 'windows_arm64',
};

const target = platformMap[`${platform}-${arch}`];
if (!target) {
  console.error(`Unsupported platform: ${platform}-${arch}`);
  process.exit(1);
}

const isWin = platform === 'win32';
const ext = isWin ? 'zip' : 'tar.gz';
const filename = `prorouter_${version}_${target}.${ext}`;
const url = `https://github.com/FernandoBolzan/ProRouter/releases/download/v${version}/${filename}`;

const binDir = path.join(__dirname, '..', 'bin');
const binaryPath = path.join(binDir, isWin ? 'prorouter.exe' : 'prorouter');

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

console.log(`Downloading ProRouter v${version} (${target})...`);

const downloadPath = path.join(binDir, filename);
const file = fs.createWriteStream(downloadPath);

https.get(url, (res) => {
  if (res.statusCode !== 200) {
    console.error(`Download failed: HTTP ${res.statusCode}`);
    console.error(`URL: ${url}`);
    console.error('Please download manually from https://github.com/FernandoBolzan/ProRouter/releases');
    file.close();
    fs.unlinkSync(downloadPath);
    process.exit(1);
  }

  res.pipe(file);
  file.on('finish', () => {
    file.close();
    try {
      if (isWin) {
        execSync(`powershell -NoProfile "Expand-Archive -Path '${downloadPath}' -DestinationPath '${binDir}' -Force"`, { stdio: 'pipe' });
        const exePath = path.join(binDir, 'prorouter.exe');
        if (fs.existsSync(exePath)) {
          fs.renameSync(exePath, binaryPath);
        }
      } else {
        execSync(`tar -xzf "${downloadPath}" -C "${binDir}"`, { stdio: 'pipe' });
      }
      fs.unlinkSync(downloadPath);
      console.log('ProRouter binary installed successfully.');
    } catch (err) {
      console.error('Extraction failed:', err.message);
      process.exit(1);
    }
  });
}).on('error', (err) => {
  console.error('Download error:', err.message);
  process.exit(1);
});

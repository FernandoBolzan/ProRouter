#!/usr/bin/env node
// Post-install script: download the correct platform binary
const https = require('https');
const fs = require('fs');
const path = require('path');
const { createGunzip } = require('zlib');
const { Extract } = require('tar-stream');
const { createWriteStream } = require('fs');

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
const url = `https://github.com/prorouter/prorouter/releases/download/v${version}/${filename}`;

const binDir = path.join(__dirname, '..', 'bin');
const binaryPath = path.join(binDir, isWin ? 'prorouter.exe' : 'prorouter');

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

console.log(`Downloading ProRouter v${version} (${target})...`);

https.get(url, (res) => {
  if (res.statusCode !== 200) {
    console.error(`Download failed: HTTP ${res.statusCode}`);
    // Create a stub that prints a helpful message
    fs.writeFileSync(binaryPath, `#!/usr/bin/env node\nconsole.log("ProRouter binary not found for ${target}. Please install via 'brew install prorouter/tap/prorouter' or download from https://github.com/prorouter/prorouter/releases");\n`);
    fs.chmodSync(binaryPath, 0o755);
    return;
  }

  if (isWin) {
    const chunks = [];
    res.on('data', (chunk) => chunks.push(chunk));
    res.on('end', () => {
      const AdmZip = require('adm-zip');
      const zip = new AdmZip(Buffer.concat(chunks));
      const entry = zip.getEntry('prorouter.exe');
      if (entry) {
        fs.writeFileSync(binaryPath, entry.getData());
        fs.chmodSync(binaryPath, 0o755);
        console.log('ProRouter binary installed.');
      }
    });
  } else {
    const gunzip = createGunzip();
    const extract = tarfs.extract();
    extract.on('entry', (header, stream, next) => {
      if (header.name === 'prorouter') {
        const ws = fs.createWriteStream(binaryPath, { mode: 0o755 });
        stream.pipe(ws);
        ws.on('finish', () => {
          console.log('ProRouter binary installed.');
          next();
        });
      } else {
        stream.resume();
        next();
      }
    });
    res.pipe(gunzip).pipe(extract);
  }
}).on('error', (err) => {
  console.error('Download error:', err.message);
});

#!/usr/bin/env node
'use strict';

const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const binDir = path.join(__dirname, '..', 'bin');
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

const targets = [
  { os: 'darwin', arch: 'arm64', out: 'seq-migrate-darwin-arm64' },
  { os: 'darwin', arch: 'amd64', out: 'seq-migrate-darwin-amd64' },
  { os: 'linux', arch: 'amd64', out: 'seq-migrate-linux-amd64' },
  { os: 'linux', arch: 'arm64', out: 'seq-migrate-linux-arm64' },
  { os: 'windows', arch: 'amd64', out: 'seq-migrate-win32-amd64.exe' }
];

console.log('⚡ Building cross-platform seq-migrate binaries...');

for (const target of targets) {
  const outputPath = path.join(binDir, target.out);
  console.log(`  -> Compiling ${target.os}/${target.arch} => bin/${target.out}...`);
  try {
    execSync(`go build -ldflags="-s -w" -o "${outputPath}" ./cmd/seq-migrate`, {
      env: {
        ...process.env,
        GOOS: target.os,
        GOARCH: target.arch,
        CGO_ENABLED: '0'
      },
      stdio: 'inherit'
    });
  } catch (err) {
    console.error(`Failed to build for ${target.os}/${target.arch}:`, err.message);
    process.exit(1);
  }
}

// Create default local binary matching host for local development
const hostOS = process.platform === 'win32' ? 'windows' : process.platform;
const hostArch = process.arch === 'x64' ? 'amd64' : process.arch;
const hostTarget = targets.find(t => t.os === hostOS && t.arch === hostArch);
if (hostTarget) {
  const src = path.join(binDir, hostTarget.out);
  const dest = path.join(binDir, process.platform === 'win32' ? 'seq-migrate.exe' : 'seq-migrate');
  fs.copyFileSync(src, dest);
  fs.chmodSync(dest, 0o755);
}

console.log('✅ All binaries built successfully.');

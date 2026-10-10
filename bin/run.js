#!/usr/bin/env node
'use strict';

const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

function getBinaryPath() {
  const platform = os.platform(); // 'darwin', 'linux', 'win32'
  const arch = os.arch();         // 'x64', 'arm64'
  const ext = platform === 'win32' ? '.exe' : '';

  // Map node arch to go arch (x64 -> amd64, arm64 -> arm64)
  const goArch = arch === 'x64' ? 'amd64' : arch;

  const candidateNames = [
    `seq-migrate-${platform}-${goArch}${ext}`,
    `seq-migrate-${platform}-${arch}${ext}`,
    `seq-migrate${ext}`
  ];

  for (const name of candidateNames) {
    const candidatePath = path.join(__dirname, name);
    if (fs.existsSync(candidatePath)) {
      return candidatePath;
    }
  }

  console.error(`Error: seq-migrate binary not found for platform: ${platform}-${arch}`);
  console.error('Please build from source using "npm run build" or "go build -o bin/seq-migrate ./cmd/seq-migrate"');
  process.exit(1);
}

const binary = getBinaryPath();
const result = spawnSync(binary, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env
});

if (result.error) {
  console.error('Error executing seq-migrate binary: ' + (result.error.message || result.error));
  process.exit(1);
}

const exitCode = result.status !== null && result.status !== undefined ? result.status : 1;
process.exit(exitCode);

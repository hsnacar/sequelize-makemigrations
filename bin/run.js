#!/usr/bin/env node
'use strict';

const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

function getBinaryPath() {
  const ext = os.platform() === 'win32' ? '.exe' : '';
  const localBinary = path.join(__dirname, 'seq-migrate' + ext);

  if (fs.existsSync(localBinary)) {
    return localBinary;
  }

  // Pre-compiled platform package fallback
  const platform = os.platform();
  const arch = os.arch();
  const platformBinary = path.join(__dirname, `seq-migrate-${platform}-${arch}` + ext);
  if (fs.existsSync(platformBinary)) {
    return platformBinary;
  }

  console.error('Error: seq-migrate binary not found for platform: ' + platform + '-' + arch);
  console.error('Please build from source using "go build -o bin/seq-migrate ./cmd/seq-migrate"');
  process.exit(1);
}

const binary = getBinaryPath();
const result = spawnSync(binary, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env
});

process.exit(result.status || 0);

'use strict';

const assert = require('assert');
const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const runJsPath = path.join(__dirname, '..', 'bin', 'run.js');

console.log('🧪 Running bin/run.js wrapper tests...');

// Test 1: run.js passes through help / version with exit code 0
{
  const res = spawnSync('node', [runJsPath, '--help'], { encoding: 'utf8' });
  assert.strictEqual(res.status, 0, 'Expected exit code 0 for --help');
  assert(res.stdout.includes('Usage: seq-migrate') || res.stdout.includes('seq-migrate'), 'Expected usage output');
  console.log('  ✅ Test 1 Passed: --help exits 0 with usage information');
}

// Test 2: run.js passes through exit code for unknown commands (exit 1)
{
  const res = spawnSync('node', [runJsPath, 'unknown-nonexistent-command'], { encoding: 'utf8' });
  assert.notStrictEqual(res.status, 0, 'Expected non-zero exit code for invalid command');
  console.log('  ✅ Test 2 Passed: Invalid command returns non-zero exit code');
}

// Test 3: run.js handles missing binary gracefully with code 1
{
  // Test by running run.js from an isolated temp dir where no binaries exist
  const tmpDir = fs.mkdtempSync(path.join(require('os').tmpdir(), 'seq-wrapper-test-'));
  const isolatedRunJs = path.join(tmpDir, 'run.js');
  fs.copyFileSync(runJsPath, isolatedRunJs);

  const res = spawnSync('node', [isolatedRunJs], { encoding: 'utf8' });
  assert.strictEqual(res.status, 1, 'Expected exit code 1 when binary is missing');
  assert(res.stderr.includes('binary not found') || res.stderr.includes('Error:'), 'Expected error message on missing binary');
  
  // Clean up
  fs.rmSync(tmpDir, { recursive: true, force: true });
  console.log('  ✅ Test 3 Passed: Missing binary exits with code 1 and descriptive error');
}

console.log('🎉 All wrapper tests passed successfully!');

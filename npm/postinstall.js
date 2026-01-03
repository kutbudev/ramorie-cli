#!/usr/bin/env node

/**
 * Ramorie CLI - postinstall script
 *
 * Downloads the correct binary for the current platform from GitHub releases.
 * Uses only Node.js built-in modules (no external dependencies).
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const VERSION = require('./package.json').version;
const REPO = 'kutbudev/ramorie-cli';

// Platform/arch mapping
const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows'
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64'
};

function download(url) {
  return new Promise((resolve, reject) => {
    const request = (url) => {
      https.get(url, (response) => {
        // Handle redirects
        if (response.statusCode === 302 || response.statusCode === 301) {
          request(response.headers.location);
          return;
        }

        if (response.statusCode !== 200) {
          reject(new Error(`Failed to download: ${response.statusCode}`));
          return;
        }

        const chunks = [];
        response.on('data', chunk => chunks.push(chunk));
        response.on('end', () => resolve(Buffer.concat(chunks)));
        response.on('error', reject);
      }).on('error', reject);
    };

    request(url);
  });
}

async function main() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    console.error(`❌ Unsupported platform: ${process.platform}/${process.arch}`);
    console.error('Please install manually from: https://github.com/kutbudev/ramorie-cli/releases');
    process.exit(0); // Don't fail npm install
  }

  const isWindows = process.platform === 'win32';
  const ext = isWindows ? 'zip' : 'tar.gz';
  // Use different name to avoid overwriting the shim script
  const binaryName = isWindows ? 'ramorie-bin.exe' : 'ramorie-bin';

  const assetName = `ramorie_${VERSION}_${platform}_${arch}.${ext}`;
  const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${assetName}`;

  const binDir = path.join(__dirname, 'bin');
  const tempDir = path.join(__dirname, 'temp-extract');
  const tempFile = path.join(__dirname, `ramorie-temp.${ext}`);
  const binaryPath = path.join(binDir, binaryName);

  console.log(`📦 Installing Ramorie v${VERSION} for ${platform}/${arch}...`);

  try {
    // Ensure directories exist
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true });
    }

    // Download the archive
    console.log(`   Downloading...`);
    const data = await download(downloadUrl);
    fs.writeFileSync(tempFile, data);
    console.log('   ✓ Downloaded');

    // Extract to temp directory (not bin!) to avoid overwriting shim
    console.log('   Extracting...');
    const extractedBinary = isWindows ? 'ramorie.exe' : 'ramorie';
    const extractedPath = path.join(tempDir, extractedBinary);

    if (isWindows) {
      execSync(`powershell -command "Expand-Archive -Path '${tempFile}' -DestinationPath '${tempDir}' -Force"`, { stdio: 'pipe' });
    } else {
      execSync(`tar -xzf "${tempFile}" -C "${tempDir}"`, { stdio: 'pipe' });
    }

    // Move only the binary to bin/ with new name
    if (fs.existsSync(extractedPath)) {
      fs.renameSync(extractedPath, binaryPath);
    }
    console.log('   ✓ Extracted');

    // Make executable (Unix only)
    if (!isWindows && fs.existsSync(binaryPath)) {
      fs.chmodSync(binaryPath, 0o755);
    }

    // Cleanup temp files and directory
    if (fs.existsSync(tempFile)) {
      fs.unlinkSync(tempFile);
    }
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    console.log(`\n✅ Ramorie v${VERSION} installed successfully!`);
    console.log('   Run "ramorie --help" to get started.\n');

  } catch (error) {
    console.error(`\n⚠️  Binary installation skipped: ${error.message}`);
    console.error('\nAlternative installation methods:');
    console.error('  • macOS/Linux: brew install kutbudev/tap/ramorie');
    console.error('  • Windows: scoop install ramorie');
    console.error('  • All: https://github.com/kutbudev/ramorie-cli/releases');

    // Don't fail the npm install
    process.exit(0);
  }
}

main();

//
// Shared, verified downloader for the agentsh core binary.
//
// Used by two callers:
//   - scripts/postinstall.js — eager fetch at install time (the happy path)
//   - bin/agentsh.js         — lazy fetch on first run, for when the postinstall
//                              never ran. npm 11.19+ withholds install scripts
//                              pending `npm install-scripts approve`, and pnpm
//                              /yarn have comparable gating, so a package that
//                              relies on postinstall alone is broken by default
//                              for a large share of users.
//
// Fails closed in both paths: a missing checksum or a mismatch aborts rather
// than leaving an unverified binary on disk. This mirrors install.sh.

const fs = require('fs');
const path = require('path');
const https = require('https');
const crypto = require('crypto');

const REPO = 'itsrohan-lang/agentsh';
const MAX_REDIRECTS = 5;

// release.yml publishes both agentsh-* and stuntdouble-* asset names; keep in sync.
const TARGETS = {
  'linux:x64': ['agentsh-linux-amd64', 'stuntdouble-linux-amd64'],
  'linux:arm64': ['agentsh-linux-arm64', 'stuntdouble-linux-arm64'],
  'darwin:x64': ['agentsh-darwin-amd64', 'stuntdouble-darwin-amd64'],
  'darwin:arm64': ['agentsh-darwin-arm64', 'stuntdouble-darwin-arm64'],
  'win32:x64': ['agentsh-windows-amd64.exe', 'stuntdouble-windows-amd64.exe'],
};

const binDir = path.resolve(__dirname, '..', 'bin');
const binaryName = process.platform === 'win32' ? 'agentsh.exe' : 'agentsh';
const binaryPath = path.join(binDir, binaryName);

class DownloadError extends Error {}

function get(url, version, redirectsLeft = MAX_REDIRECTS) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { 'User-Agent': `agentsh-npm/${version}` } }, (res) => {
        // GitHub redirects release downloads to a CDN host; follow manually.
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirectsLeft === 0) return reject(new DownloadError(`too many redirects for ${url}`));
          return resolve(get(new URL(res.headers.location, url).toString(), version, redirectsLeft - 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new DownloadError(`HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => resolve(Buffer.concat(chunks)));
        res.on('error', reject);
      })
      .on('error', reject);
  });
}

// Downloads and verifies the core binary, writing it to bin/. Resolves to the
// binary path. Throws DownloadError with an actionable message on any failure.
async function fetchBinary({ log = () => {} } = {}) {
  const pkg = require('../package.json');

  const key = `${process.platform}:${process.arch}`;
  const candidateAssets = TARGETS[key];
  if (!candidateAssets || candidateAssets.length === 0) {
    throw new DownloadError(
      `unsupported platform ${key}.\n` +
        `Supported: ${Object.keys(TARGETS).join(', ')}.\n` +
        `Build from source instead: https://github.com/${REPO}`
    );
  }

  const tag = process.env.AGENTSH_VERSION || process.env.STUNTDOUBLE_VERSION || `v${pkg.version}`;
  const base = `https://github.com/${REPO}/releases/download/${tag}`;

  let binary;
  let assetUsed;
  let lastErr;

  for (const asset of candidateAssets) {
    try {
      log(`agentsh: downloading ${asset} (${tag})...`);
      binary = await get(`${base}/${asset}`, pkg.version);
      assetUsed = asset;
      break;
    } catch (err) {
      lastErr = err;
    }
  }

  if (!binary || !assetUsed) {
    throw new DownloadError(
      `download failed for ${candidateAssets.join(' / ')} at ${base}\n` +
        `${lastErr ? lastErr.message : ''}\n` +
        `If release ${tag} has no asset for this platform, install from source or set ` +
        `AGENTSH_SKIP_DOWNLOAD=1 to bypass.`
    );
  }

  let sums;
  try {
    sums = await get(`${base}/SHA256SUMS`, pkg.version);
  } catch (err) {
    throw new DownloadError(
      `could not download SHA256SUMS for ${tag} (${err.message}). ` +
        `Refusing to install an unverified binary.`
    );
  }

  // Lines are "<sha256>  <name>" (or " *<name>" in binary mode).
  const expected = sums
    .toString('utf8')
    .split('\n')
    .map((l) => l.trim().split(/[ \t]+/))
    .filter((p) => p.length >= 2 && p[1].replace(/^\*/, '') === assetUsed)
    .map((p) => p[0])[0];

  if (!expected) {
    throw new DownloadError(`no checksum for ${assetUsed} in SHA256SUMS. Refusing to install.`);
  }

  const actual = crypto.createHash('sha256').update(binary).digest('hex');
  if (actual !== expected) {
    throw new DownloadError(
      `checksum mismatch for ${assetUsed}\n` +
        `  expected: ${expected}\n` +
        `  actual:   ${actual}\n` +
        `The download may be corrupt or tampered with. Nothing was installed.`
    );
  }

  // Write via a unique temp file and rename, so a concurrent `agentsh` invocation
  // never observes a half-written binary.
  fs.mkdirSync(binDir, { recursive: true });
  const tmp = `${binaryPath}.${process.pid}.tmp`;
  fs.writeFileSync(tmp, binary, { mode: 0o755 });
  fs.renameSync(tmp, binaryPath);

  // Link or copy backward-compatible alias binaries (ash, sd, stuntdouble)
  const ext = process.platform === 'win32' ? '.exe' : '';
  for (const alias of ['ash', 'sd', 'stuntdouble']) {
    const aliasPath = path.join(binDir, `${alias}${ext}`);
    try {
      if (fs.existsSync(aliasPath)) {
        fs.unlinkSync(aliasPath);
      }
      if (process.platform === 'win32') {
        fs.copyFileSync(binaryPath, aliasPath);
      } else {
        fs.symlinkSync(binaryName, aliasPath);
      }
    } catch {
      // Alias failure is non-fatal
    }
  }

  log(`agentsh: checksum verified, core installed (${tag}).`);
  return binaryPath;
}

module.exports = { fetchBinary, binaryPath, DownloadError, TARGETS };

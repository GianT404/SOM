import { spawnSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import { join, resolve } from 'node:path';

const root = resolve(new URL('..', import.meta.url).pathname);
const outDir = join(root, 'desktop', 'src-tauri', 'binaries');
const goCache = process.env.GOCACHE || '/tmp/som-go-build';
const target = process.argv[2] || 'linux';

const targets = {
  linux: {
    goos: 'linux',
    goarch: 'amd64',
    triple: 'x86_64-unknown-linux-gnu',
    exe: '',
    cgo: process.env.CGO_ENABLED || '1',
  },
  windows: {
    goos: 'windows',
    goarch: 'amd64',
    triple: 'x86_64-pc-windows-msvc',
    exe: '.exe',
    cgo: '0',
  },
};

const selected = target === 'all' ? Object.keys(targets) : [target];
mkdirSync(outDir, { recursive: true });

for (const name of selected) {
  const cfg = targets[name];
  if (!cfg) {
    console.error(`Unknown sidecar target "${name}". Use linux, windows, or all.`);
    process.exit(1);
  }

  const output = join(outDir, `som-backend-${cfg.triple}${cfg.exe}`);
  const result = spawnSync('go', ['build', '-trimpath', '-o', output, './cmd/server'], {
    cwd: root,
    stdio: 'inherit',
    env: {
      ...process.env,
      CGO_ENABLED: cfg.cgo,
      GOCACHE: goCache,
      GOOS: cfg.goos,
      GOARCH: cfg.goarch,
    },
  });

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }

  console.log(`Built ${output}`);
}

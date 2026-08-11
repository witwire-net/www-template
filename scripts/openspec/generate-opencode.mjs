import { spawnSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath, URL } from 'node:url';

// 端末全体のOpenSpec設定に左右されず、公式の既定値でコマンドとスキルの両方を生成する。
const isolatedConfigHome = mkdtempSync(path.join(tmpdir(), 'www-template-openspec-'));
const repositoryRoot = fileURLToPath(new URL('../..', import.meta.url));
const openspecEntry = import.meta.resolve('@fission-ai/openspec');
const openspecCli = fileURLToPath(new URL('../bin/openspec.js', openspecEntry));
const isolatedOpenSpecConfig = path.join(isolatedConfigHome, 'openspec');

// 既存生成物からの移行推測を抑止し、公式コアワークフローをコマンドとスキルの両形式で固定する。
mkdirSync(isolatedOpenSpecConfig, { recursive: true });
writeFileSync(
  path.join(isolatedOpenSpecConfig, 'config.json'),
  `${JSON.stringify({ featureFlags: {}, profile: 'core', delivery: 'both' }, null, 2)}\n`,
  'utf8'
);

let generation;

try {
  // 固定済み依存パッケージのCLIを直接実行し、OpenCode向け公式コア定義を同じ版から生成する。
  generation = spawnSync(
    process.execPath,
    [
      openspecCli,
      'init',
      '.',
      '--tools',
      'opencode',
      '--force',
      '--profile',
      'core',
      '--no-animation',
    ],
    {
      cwd: repositoryRoot,
      env: { ...process.env, XDG_CONFIG_HOME: isolatedConfigHome },
      stdio: 'inherit',
    }
  );
} finally {
  // 一時設定を必ず除去し、認証情報や端末固有設定をリポジトリへ残さない。
  rmSync(isolatedConfigHome, { recursive: true, force: true });
}

// 起動失敗とCLI失敗を呼び出し元へそのまま伝え、壊れた生成物を成功扱いしない。
if (generation.error !== undefined) {
  throw generation.error;
}

if (generation.status !== 0) {
  process.exitCode = generation.status ?? 1;
}

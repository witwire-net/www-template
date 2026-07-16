import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import process from 'node:process';

/**
 * 一時的な OpenSpec repository を作成して guard script の終了状態を検証する。
 *
 * fixture は各テスト終了時に必ず削除し、実リポジトリや他テストへ状態を残しません。
 *
 * @param {string} guardScriptPath - 実行する guard script の絶対パス。
 * @param {Record<string, string>} files - 一時リポジトリへ配置する相対パスと内容。
 * @returns {{ status: number | null; stderr: string }} guard 実行結果。
 */
export function runGuardInFixture(guardScriptPath, files) {
  const fixtureDirectory = mkdtempSync(path.join(tmpdir(), 'openspec-guard-'));

  try {
    for (const [relativePath, content] of Object.entries(files)) {
      const absolutePath = path.join(fixtureDirectory, relativePath);
      mkdirSync(path.dirname(absolutePath), { recursive: true });
      writeFileSync(absolutePath, content, 'utf8');
    }

    const result = spawnSync(process.execPath, [guardScriptPath], {
      cwd: fixtureDirectory,
      encoding: 'utf8',
    });

    return { status: result.status, stderr: result.stderr };
  } finally {
    rmSync(fixtureDirectory, { force: true, recursive: true });
  }
}

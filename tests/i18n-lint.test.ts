import { execFile } from 'node:child_process';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';

import { describe, expect, it } from 'vitest';

import { checkLocaleCatalogs } from '../scripts/i18n/check-locales';

const execFileAsync = promisify(execFile);
const repoRoot = fileURLToPath(new URL('..', import.meta.url));

interface LintMessage {
  readonly ruleId: string | null;
  readonly message: string;
}

describe('i18n lint contracts', () => {
  it('[LOCALIZATION-FE-S011] ARCH-I18N-LITERAL-GUARD は UI の直書き文言を拒否する', async () => {
    const filePath = 'packages/frontend/app/src/routes/lint-i18n/+page.svelte';
    const fullPath = path.join(repoRoot, filePath);

    await mkdir(path.dirname(fullPath), { recursive: true });
    await writeFile(fullPath, '<script lang="ts"></script>\n<p>保存</p>\n');

    try {
      const messages = await lintFile(fullPath);
      expect(
        messages.some(
          (message) => message.ruleId === 'frontend-i18n-literal-guard/no-user-facing-literals'
        )
      ).toBe(true);
    } finally {
      await rm(fullPath, { force: true });
    }
  }, 20000);

  it('[LOCALIZATION-FE-S010] ARCH-I18N-DICTIONARY-COVERAGE は辞書欠落 key と shared JSON を拒否する', async () => {
    const tempRoot = await mkdtemp(path.join(os.tmpdir(), 'i18n-check-'));

    try {
      await writeFixture(tempRoot, 'packages/web/src/lib/i18n/messages/ja/common.json', {
        greeting: 'こんにちは',
        farewell: 'さようなら',
      });
      await writeFixture(tempRoot, 'packages/web/src/lib/i18n/messages/en/common.json', {
        greeting: 'Hello',
      });
      await writeFixture(tempRoot, 'packages/frontend/i18n/src/lib/i18n/messages/ja/common.json', {
        forbidden: true,
      });

      const report = await checkLocaleCatalogs(tempRoot);

      expect(report.complete).toBe(false);
      expect(report.issues).toEqual([
        {
          surface: 'web',
          locale: 'en',
          namespace: 'common',
          missingKeys: ['farewell'],
        },
      ]);
      expect(report.forbiddenFiles).toEqual([
        {
          filePath: 'packages/frontend/i18n/src/lib/i18n/messages/ja/common.json',
        },
      ]);
    } finally {
      await rm(tempRoot, { recursive: true, force: true });
    }
  }, 20000);
});

async function lintFile(fullPath: string): Promise<LintMessage[]> {
  const eslintArgs = ['exec', 'eslint', '--format', 'json', fullPath];
  let stdout = '';

  try {
    const result = await execFileAsync('pnpm', eslintArgs, {
      cwd: repoRoot,
      maxBuffer: 10 * 1024 * 1024,
    });
    stdout = result.stdout;
  } catch (error) {
    const lintError = error as { stdout?: string | Buffer };
    stdout =
      typeof lintError.stdout === 'string'
        ? lintError.stdout
        : (lintError.stdout?.toString() ?? '');
    if (stdout === '') {
      throw error;
    }
  }

  const parsed = JSON.parse(stdout) as { readonly messages: LintMessage[] }[];
  return parsed.at(0)?.messages ?? [];
}

async function writeFixture(
  rootDir: string,
  relativePath: string,
  content: unknown
): Promise<void> {
  const fullPath = path.join(rootDir, relativePath);
  await mkdir(path.dirname(fullPath), { recursive: true });
  await writeFile(fullPath, `${JSON.stringify(content, null, 2)}\n`);
}

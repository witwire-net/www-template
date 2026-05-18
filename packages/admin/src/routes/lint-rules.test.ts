import { execFile } from 'node:child_process';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';

import { describe, expect, it } from 'vitest';

const sourceDir = fileURLToPath(new URL('.', import.meta.url));
const adminRoot = path.resolve(sourceDir, '..', '..');
const repoRoot = path.resolve(adminRoot, '..', '..');
const execFileAsync = promisify(execFile);

interface LintCase {
  name: string;
  filePath: string;
  source: string;
  ruleId: string;
}

describe('Admin ESLint lint-rule contracts', () => {
  for (const testCase of lintCases) {
    it(
      testCase.name,
      async () => {
        // ESLint CLI を別プロセスで起動し、実在 fixture を 1 件だけ lint して ruleId を確認する。
        const messages = await lintText(testCase.filePath, testCase.source);
        expectViolation(messages, testCase.ruleId);
      },
      20000
    );
  }
});

const lintCases: LintCase[] = [
  {
    name: '19.1 ハードコードされた DB 接続文字列は lint エラーになる',
    filePath: 'packages/admin/src/lib/server/infrastructure/config/lint-hardcoded-db.ts',
    source: "export const databaseUrl = 'postgres://user:pass@host:5432/db';",
    ruleId: 'admin-security/no-hardcoded-db-strings',
  },
  {
    name: '19.2 @html ディレクティブは lint エラーになる',
    filePath: 'packages/admin/src/routes/lint-at-html/+page.svelte',
    source: `<script lang="ts">\n\tconst content = '<p>admin</p>';\n</script>\n{@html content}`,
    ruleId: 'svelte/no-at-html-tags',
  },
  {
    name: '19.3 SQL テンプレートリテラルは lint エラーになる',
    filePath: 'packages/admin/src/lib/server/models/lint-sql-template.ts',
    source: 'export const query = `select * from public.accounts`;',
    ruleId: 'admin-security/no-sql-template-literals',
  },
  {
    name: '19.3a Prisma unsafe raw query は lint エラーになる',
    filePath: 'packages/admin/src/lib/server/models/lint-prisma-unsafe.ts',
    source:
      'export const run = (prisma: { $queryRawUnsafe: (...args: readonly unknown[]) => unknown }) => prisma.$queryRawUnsafe(`select * from public.accounts`);',
    ruleId: 'admin-security/no-raw-unsafe',
  },
  {
    name: '19.4 Model から services を import すると lint エラーになる',
    filePath: 'packages/admin/src/lib/server/models/foo/lint-model-import.ts',
    source: "import { searchAccounts } from '../../services/accounts/search';",
    ruleId: 'boundaries/element-types',
  },
  {
    name: '19.5 Service から components を import すると lint エラーになる',
    filePath: 'packages/admin/src/lib/server/services/foo/lint-service-import.ts',
    source: "import AdminShell from '../../../components/layout/AdminShell.svelte';",
    ruleId: 'boundaries/element-types',
  },
  {
    name: '19.6 Admin から @www-template/api を import すると lint エラーになる',
    filePath: 'packages/admin/src/lib/server/services/foo/lint-admin-api-import.ts',
    source: "import { statusApi } from '@www-template/api';",
    ruleId: 'boundaries/element-types',
  },
  {
    name: '19.7 View から Model を import すると lint エラーになる',
    filePath: 'packages/admin/src/lib/components/layout/foo/lint-view-import-model.svelte',
    source: `<script lang="ts">\n\timport { findOperatorById } from '../../../server/models/operators';\n</script>`,
    ruleId: 'boundaries/element-types',
  },
  {
    name: '19.8 Admin から api を import すると lint エラーになる',
    filePath: 'packages/admin/src/routes/lint-admin-api-import/+page.svelte',
    source: `<script lang="ts">\n\timport { statusApi } from '@www-template/api';\n</script>`,
    ruleId: 'no-restricted-imports',
  },
];

function expectViolation(messages: LintMessage[], ruleId: string): void {
  // 期待する ruleId が 1 件以上存在することだけを確認し、fixture が別のルールで偶然落ちる誤判定を避ける。
  const hit = messages.find((message) => message.ruleId === ruleId);
  expect(hit).toBeDefined();
}

async function lintText(filePath: string, source: string): Promise<LintMessage[]> {
  // TypeScript project service が実在ファイルを必要とするため、repo 配下に一時 fixture を作ってから ESLint CLI で lint する。
  const fullPath = path.join(repoRoot, filePath);
  await mkdir(path.dirname(fullPath), { recursive: true });
  await writeFile(fullPath, source);

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
  } finally {
    await rm(fullPath, { force: true });
  }

  const parsed = JSON.parse(stdout) as { filePath: string; messages: LintMessage[] }[];
  return parsed.at(0)?.messages ?? [];
}

interface LintMessage {
  ruleId: string | null;
  message: string;
}

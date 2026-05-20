import { readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

import {
  ADMIN_OPERATOR_LOCALES,
  DEFAULT_OPERATOR_LOCALE,
  parseOperatorLocale,
} from './operator_locale.js';

const adminSrcRoot = path.resolve(fileURLToPath(new URL('../../../', import.meta.url)));

describe('models/operator_locale', () => {
  it('Admin operator locale は ja/en だけを受け付ける', () => {
    // Admin package-local の locale validator が Product AccountSetting に依存せず対応値だけを返すことを確認する。
    expect(DEFAULT_OPERATOR_LOCALE).toBe('ja');
    expect(parseOperatorLocale('ja')).toBe('ja');
    expect(parseOperatorLocale('en')).toBe('en');
    expect(() => parseOperatorLocale('fr')).toThrow('unsupported admin operator locale');
  });

  it('ARCH-ADMIN-LOCALE-INDEPENDENCE Admin locale 実装は Product AccountSetting と generated Product API を import しない', async () => {
    // Admin operator locale は Admin package-local symbols で扱い、Product TypeSpec/generated SDK/Product AccountSetting を参照しない。
    const sources = await readAdminSources(adminSrcRoot);
    const importStatements = sources.flatMap((source) => extractImportStatements(source.content));
    const combinedImports = importStatements.join('\n');
    expect(combinedImports).not.toMatch(/Account(?:Setting|Locale)/);
    expect(combinedImports).not.toContain('@www-template/api');
    expect(combinedImports).not.toContain('packages/typespec');
    expect(combinedImports).not.toContain('packages/frontend/api');
  });

  it('ARCH-ADMIN-LOCALE-INDEPENDENCE Admin locale JSON は ja/en に揃い TS 辞書を持たない', async () => {
    // Admin operator locale、Admin-owned JSON catalog、package-local i18n entrypoint の対応 locale を ja/en に固定する。
    const i18nRoot = path.resolve(adminSrcRoot, 'lib/i18n');
    const messageRoot = path.join(i18nRoot, 'messages');
    const localeDirectories = (await readdir(messageRoot, { withFileTypes: true }))
      .filter((entry) => entry.isDirectory())
      .map((entry) => entry.name)
      .sort();

    expect(localeDirectories).toEqual([...ADMIN_OPERATOR_LOCALES].sort());
    expect(await existsFile(path.join(messageRoot, 'ja', 'common.json'))).toBe(true);
    expect(await existsFile(path.join(messageRoot, 'en', 'common.json'))).toBe(true);
    expect(await existsFile(path.join(i18nRoot, 'index.ts'))).toBe(true);
    expect(
      await existsFile(path.resolve(adminSrcRoot, 'lib/server/infrastructure/i18n/catalogs.ts'))
    ).toBe(false);
  });

  it('ARCH-ADMIN-I18N-SHARED-RUNTIME Admin i18n runtime は shared frontend i18n core を利用する', async () => {
    // Admin i18n runtime は flatten / interpolate の standalone 実装を持たず、shared frontend i18n core に委譲する。
    const runtimeSource = await readFile(path.join(adminSrcRoot, 'lib/i18n/runtime.ts'), 'utf8');

    expect(runtimeSource).toContain("from '@www-template/i18n'");
    expect(runtimeSource).toContain('createTranslator(');
    expect(runtimeSource).toContain('defineI18nConfig(');
    expect(runtimeSource).toContain('loadJsonCatalog(');
    expect(runtimeSource).not.toContain('flattenCatalog(');
    expect(runtimeSource).not.toContain('interpolateTemplate(');
  });
});

async function readAdminSources(root: string): Promise<{ filePath: string; content: string }[]> {
  // Admin src 配下を再帰的に読み、test 自身や生成物ではなく実装 source の境界違反だけを検出する。
  const entries = await readdir(root, { withFileTypes: true });
  const result: { filePath: string; content: string }[] = [];
  for (const entry of entries) {
    const nextPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      result.push(...(await readAdminSources(nextPath)));
      continue;
    }
    if (/\.(ts|svelte)$/.exec(entry.name) === null || entry.name.endsWith('.test.ts')) {
      continue;
    }
    result.push({ filePath: nextPath, content: await readFile(nextPath, 'utf8') });
  }
  return result;
}

function extractImportStatements(content: string): string[] {
  // コメントやテスト説明文ではなく実際の import 文だけを検査し、アーキテクチャ依存の混入を誤検知なく捉える。
  const matches = content.matchAll(/import(?:\s+type)?[\S\s]*?from\s+["']([^"']+)["']/g);
  return [...matches].map((match) => match[0]);
}

async function existsFile(filePath: string): Promise<boolean> {
  // 辞書配置 guard は存在確認だけを行い、生成物や別 package へ探索範囲を広げない。
  try {
    const stats = await stat(filePath);
    return stats.isFile();
  } catch {
    return false;
  }
}

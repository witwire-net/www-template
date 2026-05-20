import { readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

import {
  getCatalogCoverage,
  loadJsonCatalog,
  type CatalogTree,
  type Locale,
} from '../../packages/frontend/i18n/src/index';

/**
 * locale 検証で扱う surface 名です。
 */
export type LocaleSurface = 'web' | 'app' | 'admin' | 'ui';

/**
 * 辞書 coverage の 1 件を表します。
 */
export interface LocaleCoverageIssue {
  /**
   * 対象の surface です。
   */
  readonly surface: LocaleSurface;

  /**
   * 欠落が見つかった locale です。
   */
  readonly locale: Locale;

  /**
   * 対象 namespace です。
   */
  readonly namespace: string;

  /**
   * その locale に存在しない key path です。
   */
  readonly missingKeys: readonly string[];
}

/**
 * 禁止される共有 i18n package 内の JSON file を表します。
 */
export interface ForbiddenLocaleJsonIssue {
  /**
   * 見つかった file path です。
   */
  readonly filePath: string;
}

/**
 * locale 検証の結果です。
 */
export interface LocaleCheckReport {
  /**
   * すべての surface で key coverage が一致し、禁止 file も無い場合に `true` になります。
   */
  readonly complete: boolean;

  /**
   * surface ごとの key coverage 差分です。
   */
  readonly issues: readonly LocaleCoverageIssue[];

  /**
   * `packages/frontend/i18n` 配下で見つかった禁止 JSON file です。
   */
  readonly forbiddenFiles: readonly ForbiddenLocaleJsonIssue[];
}

const SUPPORTED_LOCALES: readonly Locale[] = ['ja', 'en'];
const SURFACES: readonly { readonly surface: LocaleSurface; readonly root: string }[] = [
  { surface: 'web', root: 'packages/web/src' },
  { surface: 'app', root: 'packages/frontend/app/src' },
  { surface: 'admin', root: 'packages/admin/src' },
  { surface: 'ui', root: 'packages/frontend/ui/src' },
];

const SHARED_I18N_ROOT = 'packages/frontend/i18n/src';
const LOCALE_SEGMENTS = new Set<string>(SUPPORTED_LOCALES);

const toPosix = (value: string): string => value.split(path.sep).join('/');

const isJsonFile = (filePath: string): boolean => filePath.endsWith('.json');

async function collectFiles(rootDir: string): Promise<string[]> {
  const entries = await readdir(rootDir, { withFileTypes: true });
  const files: string[] = [];

  for (const entry of entries) {
    const absolutePath = path.join(rootDir, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await collectFiles(absolutePath)));
      continue;
    }

    if (entry.isFile()) {
      files.push(absolutePath);
    }
  }

  return files;
}

const parseLocaleFromFilePath = (
  surfaceRoot: string,
  filePath: string
): { readonly locale: Locale; readonly namespace: string } | null => {
  const relativePath = path.relative(surfaceRoot, filePath);
  const segments = relativePath.split(path.sep);
  const localeIndex = segments.findIndex((segment) => LOCALE_SEGMENTS.has(segment));

  if (localeIndex === -1 || localeIndex === segments.length - 1) {
    return null;
  }

  const localeSegment = segments.at(localeIndex);
  if (localeSegment !== 'ja' && localeSegment !== 'en') {
    return null;
  }
  const locale: Locale = localeSegment;
  const namespaceSegments = segments.slice(localeIndex + 1);
  const lastSegment = namespaceSegments.at(-1);
  if (lastSegment === undefined) {
    return null;
  }
  if (!lastSegment.endsWith('.json')) {
    return null;
  }

  const namespace = namespaceSegments.join('/').replace(/\.json$/u, '');
  return { locale, namespace };
};

const collectSurfaceCatalogs = async (
  repoRoot: string,
  surfaceRoot: string
): Promise<Record<Locale, Record<string, CatalogTree>>> => {
  const absoluteSurfaceRoot = path.join(repoRoot, surfaceRoot);
  if (!(await existsDir(absoluteSurfaceRoot))) {
    return Object.fromEntries(
      SUPPORTED_LOCALES.map((locale) => [locale, Object.create(null)])
    ) as Record<Locale, Record<string, CatalogTree>>;
  }

  const catalogs = Object.fromEntries(
    SUPPORTED_LOCALES.map((locale) => [locale, Object.create(null)])
  ) as Record<Locale, Record<string, CatalogTree>>;

  for (const filePath of await collectFiles(absoluteSurfaceRoot)) {
    if (!isJsonFile(filePath)) {
      continue;
    }

    const parsed = parseLocaleFromFilePath(absoluteSurfaceRoot, filePath);
    if (parsed === null) {
      continue;
    }

    const payload = JSON.parse(await readFile(filePath, 'utf8')) as unknown;
    catalogs[parsed.locale][parsed.namespace] = loadJsonCatalog(payload);
  }

  return catalogs;
};

const existsDir = async (filePath: string): Promise<boolean> => {
  try {
    const stats = await stat(filePath);
    return stats.isDirectory();
  } catch {
    return false;
  }
};

/**
 * 共有 i18n package 内に locale JSON が混ざっていないかを検証します。
 */
export async function findForbiddenLocaleJsonFiles(
  repoRoot: string
): Promise<ForbiddenLocaleJsonIssue[]> {
  const sharedRoot = path.join(repoRoot, SHARED_I18N_ROOT);
  if (!(await existsDir(sharedRoot))) {
    return [];
  }

  const files = await collectFiles(sharedRoot);
  return files
    .filter(isJsonFile)
    .map((filePath) => ({ filePath: toPosix(path.relative(repoRoot, filePath)) }));
}

/**
 * surface ごとの locale JSON key coverage を検証します。
 *
 * `packages/web`、`packages/frontend/app`、`packages/admin`、`packages/frontend/ui` の
 * `ja` / `en` 辞書差分を集約し、欠落 key を一覧化します。
 */
export async function checkLocaleCatalogs(repoRoot: string): Promise<LocaleCheckReport> {
  const issues: LocaleCoverageIssue[] = [];

  for (const { surface, root } of SURFACES) {
    const catalogs = await collectSurfaceCatalogs(repoRoot, root);
    const report = getCatalogCoverage(catalogs);
    for (const issue of report.issues) {
      issues.push({
        surface,
        locale: issue.locale,
        namespace: issue.namespace,
        missingKeys: issue.missingKeys,
      });
    }
  }

  const forbiddenFiles = await findForbiddenLocaleJsonFiles(repoRoot);

  return Object.freeze({
    complete: issues.length === 0 && forbiddenFiles.length === 0,
    issues: Object.freeze(issues),
    forbiddenFiles: Object.freeze(forbiddenFiles),
  });
}

/**
 * 検証結果を lint 向けの単一メッセージに整形します。
 */
export function formatLocaleCheckReport(report: LocaleCheckReport): string {
  const lines: string[] = [];

  if (report.issues.length > 0) {
    lines.push('ARCH-I18N-DICTIONARY-COVERAGE: locale key coverage が不完全です。');
    for (const issue of report.issues) {
      lines.push(
        `- ${issue.surface}:${issue.locale}:${issue.namespace} -> ${issue.missingKeys.join(', ')}`
      );
    }
  }

  if (report.forbiddenFiles.length > 0) {
    lines.push(
      'ARCH-I18N-FORBIDDEN-JSON: packages/frontend/i18n 配下に locale JSON file が存在します。'
    );
    for (const issue of report.forbiddenFiles) {
      lines.push(`- ${issue.filePath}`);
    }
  }

  return lines.join('\n');
}

/**
 * CLI entrypoint です。
 */
export async function main(argv: readonly string[] = process.argv.slice(2)): Promise<number> {
  const repoRoot = path.resolve(argv[0] ?? process.cwd());
  const report = await checkLocaleCatalogs(repoRoot);

  if (report.complete) {
    return 0;
  }

  const message = formatLocaleCheckReport(report);
  if (message.length > 0) {
    process.stderr.write(`${message}\n`);
  }

  return 1;
}

const isMain =
  process.argv[1] !== undefined &&
  import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href;

if (isMain) {
  void main().then((exitCode) => {
    process.exitCode = exitCode;
  });
}

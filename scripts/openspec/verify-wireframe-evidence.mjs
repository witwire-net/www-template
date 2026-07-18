import { Buffer } from 'node:buffer';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, realpathSync, statSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

import { check as checkPrettier, resolveConfig } from 'prettier';

const SCRIPT_DIRECTORY = path.dirname(fileURLToPath(import.meta.url));
const REPOSITORY_ROOT = path.resolve(SCRIPT_DIRECTORY, '..', '..');
const OPENSPEC_CHANGES_DIRECTORY = path.join(REPOSITORY_ROOT, 'openspec', 'changes');
const GENERATOR_PATH = path.join(
  REPOSITORY_ROOT,
  '.opencode',
  'skills',
  'wireframe',
  'scripts',
  'generate-preview.mjs'
);
const JSON_SUFFIX = '.wireframe.json';
const HTML_SUFFIX = '.wireframe.html';
const SCREENSHOT_SUFFIX = '.wireframe-screenshot.png';
const PNG_SIGNATURE = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
const PNG_IHDR_CHUNK_TYPE = Buffer.from('IHDR', 'ascii');

/**
 * 検査対象が指定ディレクトリの子孫に収まるかを判定する。
 *
 * @param {string} parentPath - 境界となるディレクトリの絶対パス。
 * @param {string} candidatePath - 境界内であることを確認する絶対パス。
 * @returns {boolean} candidate が parent の子孫である場合は `true`。
 */
function isPathInside(parentPath, candidatePath) {
  const relativePath = path.relative(parentPath, candidatePath);
  return (
    relativePath.length > 0 &&
    relativePath !== '..' &&
    !relativePath.startsWith(`..${path.sep}`) &&
    !path.isAbsolute(relativePath)
  );
}

/**
 * OpenSpec artifactを実体パスまで解決し、範囲外参照と非通常ファイルを拒否する。
 *
 * lexical pathとreal pathの双方を検査するため、active Change内のsymlinkを経由して
 * repository外のファイルを証跡へ混入させることはできない。
 *
 * @param {string} candidatePath - 検査対象artifactのパス。
 * @param {string} artifactLabel - エラーに表示するartifact種別。
 * @returns {{ absolutePath: string; realPath: string; stats: import('node:fs').Stats }} 解決済みartifact。
 * @throws {Error} artifactが存在しない、通常ファイルでない、またはOpenSpec範囲外の場合。
 */
function resolveOpenSpecArtifact(candidatePath, artifactLabel) {
  const absolutePath = path.resolve(candidatePath);
  const absoluteChangesDirectory = path.resolve(OPENSPEC_CHANGES_DIRECTORY);

  if (!isPathInside(absoluteChangesDirectory, absolutePath)) {
    throw new Error(`${artifactLabel} は openspec/changes 配下に限定されます: ${candidatePath}`);
  }

  let realChangesDirectory;
  let realPath;
  try {
    realChangesDirectory = realpathSync(absoluteChangesDirectory);
    realPath = realpathSync(absolutePath);
  } catch {
    throw new Error(`${artifactLabel} が存在しません: ${absolutePath}`);
  }

  if (!isPathInside(realChangesDirectory, realPath)) {
    throw new Error(
      `${artifactLabel} の symlink が openspec/changes の外を参照しています: ${absolutePath}`
    );
  }

  const stats = statSync(realPath);
  if (!stats.isFile()) {
    throw new Error(`${artifactLabel} は通常ファイルである必要があります: ${absolutePath}`);
  }

  return { absolutePath, realPath, stats };
}

/**
 * wireframe JSONから同一画面のgenerated previewとscreenshotの規定パスを導出する。
 *
 * @param {string} sourcePath - 検査対象`.wireframe.json`のパス。
 * @returns {{ sourcePath: string; previewPath: string; screenshotPath: string }} 画面単位のartifactパス。
 * @throws {Error} sourceがactive Changeの標準配置に従っていない場合。
 */
function getArtifactPaths(sourcePath) {
  const absoluteSourcePath = path.resolve(sourcePath);
  const relativeSourcePath = path.relative(OPENSPEC_CHANGES_DIRECTORY, absoluteSourcePath);
  const pathSegments = relativeSourcePath.split(path.sep);

  if (
    pathSegments.length !== 3 ||
    pathSegments[0] === 'archive' ||
    pathSegments[1] !== 'wireframes' ||
    !pathSegments[2].endsWith(JSON_SUFFIX)
  ) {
    throw new Error(
      `検証対象は openspec/changes/<change-id>/wireframes/<screen>${JSON_SUFFIX} に配置してください: ${sourcePath}`
    );
  }

  const screenSlug = pathSegments[2].slice(0, -JSON_SUFFIX.length);
  const changeDirectory = path.join(OPENSPEC_CHANGES_DIRECTORY, pathSegments[0]);
  return {
    sourcePath: absoluteSourcePath,
    previewPath: `${absoluteSourcePath.slice(0, -JSON_SUFFIX.length)}${HTML_SUFFIX}`,
    screenshotPath: path.join(
      changeDirectory,
      'wireframe-screenshots',
      `${screenSlug}${SCREENSHOT_SUFFIX}`
    ),
  };
}

/**
 * PNG signatureとIHDRの寸法を読み、画像証跡として成立していることを確認する。
 *
 * @param {Buffer} screenshot - screenshot PNGのバイナリ。
 * @param {string} screenshotPath - エラー表示用のscreenshotパス。
 * @returns {{ width: number; height: number }} PNGのpixel寸法。
 * @throws {Error} PNG signature、IHDR、または寸法が不正な場合。
 */
function readPngDimensions(screenshot, screenshotPath) {
  if (
    screenshot.length < 24 ||
    !screenshot.subarray(0, PNG_SIGNATURE.length).equals(PNG_SIGNATURE) ||
    !screenshot.subarray(12, 16).equals(PNG_IHDR_CHUNK_TYPE)
  ) {
    throw new Error(`wireframe screenshot が PNG ではありません: ${screenshotPath}`);
  }

  const width = screenshot.readUInt32BE(16);
  const height = screenshot.readUInt32BE(20);
  if (width === 0 || height === 0) {
    throw new Error(`wireframe screenshot の寸法が不正です: ${screenshotPath}`);
  }

  return { width, height };
}

/**
 * artifactの内容識別子とfilesystem metadataを副作用なしで生成する。
 *
 * @param {{ absolutePath: string; realPath: string; stats: import('node:fs').Stats }} artifact - 解決済みartifact。
 * @returns {{ path: string; sha256: string; inode: number; bytes: number; modifiedAt: string }} 検証報告値。
 */
function describeArtifact(artifact) {
  return {
    path: path.relative(REPOSITORY_ROOT, artifact.absolutePath).split(path.sep).join('/'),
    sha256: createHash('sha256').update(readFileSync(artifact.realPath)).digest('hex'),
    inode: artifact.stats.ino,
    bytes: artifact.stats.size,
    modifiedAt: artifact.stats.mtime.toISOString(),
  };
}

/**
 * 汎用wireframe generatorへpreview drift検査を委譲する。
 *
 * @param {string} sourcePath - 検査対象wireframe JSONの絶対パス。
 * @returns {void}
 * @throws {Error} generatorがpreview欠落またはdriftを検出した場合。
 */
function verifyPreview(sourcePath) {
  const result = spawnSync(process.execPath, [GENERATOR_PATH, '--check', sourcePath], {
    cwd: REPOSITORY_ROOT,
    encoding: 'utf8',
  });

  if (result.status !== 0) {
    const detail = `${result.stdout}${result.stderr}`.trim();
    throw new Error(detail || `generated previewを検証できません: ${sourcePath}`);
  }
}

/**
 * 1画面分のOpenSpec wireframe source、preview、screenshotをread-onlyで検証する。
 *
 * @param {string} sourcePath - `.wireframe.json` sourceのパス。
 * @returns {Promise<{ source: ReturnType<typeof describeArtifact>; preview: ReturnType<typeof describeArtifact>; screenshot: ReturnType<typeof describeArtifact>; dimensions: { width: number; height: number } }>} 検証済み証跡。
 * @throws {Error} 配置、整形、preview drift、PNG、またはpath境界に問題がある場合。
 */
async function verifyArtifactSet(sourcePath) {
  const artifactPaths = getArtifactPaths(sourcePath);
  const source = resolveOpenSpecArtifact(artifactPaths.sourcePath, 'wireframe JSON');
  const preview = resolveOpenSpecArtifact(artifactPaths.previewPath, 'generated preview');
  const screenshot = resolveOpenSpecArtifact(artifactPaths.screenshotPath, 'wireframe screenshot');

  // repositoryのPrettier設定でsourceを検査し、証跡検証中はファイルを書き換えません。
  const prettierConfig = await resolveConfig(source.absolutePath);
  const formatted = await checkPrettier(readFileSync(source.realPath, 'utf8'), {
    ...(prettierConfig ?? {}),
    filepath: source.absolutePath,
  });
  if (!formatted) {
    throw new Error(`wireframe JSON が Prettier 形式ではありません: ${source.absolutePath}`);
  }

  // preview生成規則は汎用wireframe generatorだけに置き、OpenSpec側では再実装しません。
  verifyPreview(source.absolutePath);

  // 視覚確認済みのscreenshotが実在するPNGであり、有効な寸法を持つことを保証します。
  const dimensions = readPngDimensions(readFileSync(screenshot.realPath), screenshot.absolutePath);
  return {
    source: describeArtifact(source),
    preview: describeArtifact(preview),
    screenshot: describeArtifact(screenshot),
    dimensions,
  };
}

/**
 * 1 artifact分のmetadataを型が確定した形式でレポートへ追加する。
 *
 * @param {string[]} lines - 追記先のレポート行。
 * @param {string} label - source、preview、screenshotの種別。
 * @param {ReturnType<typeof describeArtifact>} artifact - 出力対象metadata。
 * @returns {void}
 */
function appendArtifactReport(lines, label, artifact) {
  lines.push(
    `  ${label}: ${artifact.path} sha256=${artifact.sha256} inode=${String(artifact.inode)} bytes=${String(artifact.bytes)} mtime=${artifact.modifiedAt}`
  );
}

/**
 * CLI引数の全sourceを検証し、全件成功した場合だけ証跡レポートを出力する。
 *
 * @returns {Promise<void>} 完了通知。失敗は`process.exitCode`とstderrで表現する。
 */
async function main() {
  const sourcePaths = process.argv.slice(2);
  if (sourcePaths.length === 0) {
    process.stderr.write(
      'Usage: node scripts/openspec/verify-wireframe-evidence.mjs <source.wireframe.json>...\n'
    );
    process.exitCode = 1;
    return;
  }

  const reports = [];
  const errors = [];
  for (const sourcePath of sourcePaths) {
    try {
      reports.push(await verifyArtifactSet(sourcePath));
    } catch (error) {
      errors.push(error instanceof Error ? error.message : String(error));
    }
  }

  if (errors.length > 0) {
    process.stderr.write(
      `OpenSpec wireframe evidence verification failed:\n${errors.map((error) => `- ${error}`).join('\n')}\n`
    );
    process.exitCode = 1;
    return;
  }

  const lines = ['OpenSpec wireframe evidence verification: PASS'];
  for (const report of reports) {
    lines.push(`- screen: ${report.source.path}`);
    lines.push(
      `  dimensions: ${String(report.dimensions.width)}x${String(report.dimensions.height)}`
    );
    appendArtifactReport(lines, 'source', report.source);
    appendArtifactReport(lines, 'preview', report.preview);
    appendArtifactReport(lines, 'screenshot', report.screenshot);
  }
  process.stdout.write(`${lines.join('\n')}\n`);
}

await main();

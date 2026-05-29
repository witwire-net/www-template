#!/usr/bin/env node

import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const packageRoot = fileURLToPath(new URL('..', import.meta.url));
const productRouteRoot = join(packageRoot, 'src', 'routes', 'v1');

// Product surface の TypeSpec source が Admin route namespace を import / 参照しないことを表す禁止語彙。
// Admin model は共有 read model として存在できるが、Admin.ApiV1 route namespace は Product artifact へ混入してはならない。
const productSurfaceForbiddenPatterns = [
  {
    pattern: /import\s+["'][^"']*routes\/admin-v1\/[^"']*["'];?/u,
    reason: 'Product route source must not import Admin route files',
  },
  {
    pattern: /import\s+["'][^"']*admin-v1\/[^"']*["'];?/u,
    reason: 'Product route source must not import Admin route namespace files',
  },
  {
    pattern: /\bAdmin\.ApiV1\b/u,
    reason: 'Product route source must not reference Admin.ApiV1 namespace',
  },
];

const explicitInputFiles = process.argv.slice(2);

// Step 1: CLI 引数がある場合は test fixture などの明示 path だけを検査し、通常実行では Product route tree 全体を検査する。
const inputFiles = explicitInputFiles.length > 0 ? explicitInputFiles : collectTypeSpecFiles(productRouteRoot);

// Step 2: 各 TypeSpec source を行単位で評価し、違反箇所を path:line 付きで蓄積する。
const violations = inputFiles.flatMap((filePath) => detectProductSurfaceBoundaryViolations(filePath));

// Step 3: 1 件でも違反があれば stderr にすべて出し、contract lint の fail-closed な終了コードにする。
if (violations.length > 0) {
  for (const violation of violations) {
    console.error(violation);
  }
  process.exit(1);
}

/**
 * collectTypeSpecFiles は指定 directory 配下の `.tsp` source を再帰的に列挙する。
 *
 * @param {string} directory 検査対象の directory。Product route namespace の root を渡す。
 * @returns {string[]} 検査対象 TypeSpec file の絶対 path 一覧。directory の副作用はない。
 * @throws {Error} directory が読めない場合は Node.js の filesystem error をそのまま送出する。
 */
function collectTypeSpecFiles(directory) {
  // Step 1: directory entry を type 付きで読み、file と subdirectory を確実に分ける。
  const entries = readdirSync(directory, { withFileTypes: true });
  const files = [];

  // Step 2: subdirectory は再帰し、`.tsp` file だけを検査対象として返す。
  for (const entry of entries) {
    const entryPath = join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...collectTypeSpecFiles(entryPath));
      continue;
    }
    if (entry.isFile() && entry.name.endsWith('.tsp')) {
      files.push(entryPath);
    }
  }

  return files;
}

/**
 * detectProductSurfaceBoundaryViolations は Product TypeSpec source 内の Admin route namespace 混入を検出する。
 *
 * @param {string} filePath 検査する TypeSpec file。通常は Product route 配下の `.tsp` file または test fixture を渡す。
 * @returns {string[]} `path:line` 付きの違反 message。一覧が空なら Product/Admin route 境界は保たれている。
 * @throws {Error} file が読めない場合は Node.js の filesystem error をそのまま送出する。
 */
function detectProductSurfaceBoundaryViolations(filePath) {
  // Step 1: source を UTF-8 text として読み、検査対象 file 以外へ副作用を出さない。
  const source = readFileSync(filePath, 'utf8');
  const lines = source.split(/\r?\n/u);
  const displayPath = relative(process.cwd(), filePath);
  const violations = [];

  // Step 2: 行ごとに禁止 pattern を適用し、どの Admin route namespace 参照が混入したかを説明する。
  for (const [index, line] of lines.entries()) {
    const executableLine = stripLineComment(line);
    for (const { pattern, reason } of productSurfaceForbiddenPatterns) {
      if (pattern.test(executableLine)) {
        violations.push(`${displayPath}:${index + 1}: ${reason}`);
      }
    }
  }

  return violations;
}

/**
 * stripLineComment は TypeSpec の `//` comment を行末から除去する。
 *
 * @param {string} line TypeSpec source の 1 行。
 * @returns {string} comment を除いた検査用 text。文字列 literal 内の `//` は fixture で使わない前提の軽量 guardrail 用処理。
 */
function stripLineComment(line) {
  // Step 1: comment の説明文で Admin.ApiV1 と書いた場合に誤検知しないよう、実行部分だけを残す。
  const commentIndex = line.indexOf('//');
  if (commentIndex === -1) {
    return line;
  }

  return line.slice(0, commentIndex);
}

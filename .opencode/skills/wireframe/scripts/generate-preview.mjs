import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const SCRIPT_DIRECTORY = path.dirname(fileURLToPath(import.meta.url));
const REPOSITORY_ROOT = path.resolve(SCRIPT_DIRECTORY, '..', '..', '..', '..');
const TEMPLATE_PATH = path.join(
  REPOSITORY_ROOT,
  '.opencode',
  'skills',
  'wireframe',
  'wireframe-template.html'
);
const TEMPLATE_MARKER = 'const WIREFRAME_DATA = null; // %%WIREFRAME_DATA%%';
const JSON_SUFFIX = '.wireframe.json';
const HTML_SUFFIX = '.wireframe.html';

/**
 * CLI 引数を解析し、preview を書き込むか drift を検査するかを決定する。
 *
 * @param {string[]} arguments_ - `node` と script path を除いた CLI 引数。
 * @returns {{ check: boolean; outputPath: string | null; sourcePaths: string[] }} 実行条件。
 * @throws {Error} 未知の option や不完全な `--output` 指定がある場合。
 */
function parseArguments(arguments_) {
  let check = false;
  let outputPath = null;
  /** @type {string[]} */
  const sourcePaths = [];

  let outputPathExpected = false;
  for (const argument of arguments_) {
    if (outputPathExpected) {
      if (argument.startsWith('--')) {
        throw new Error('`--output` の後に preview HTML の出力パスを指定してください。');
      }
      outputPath = argument;
      outputPathExpected = false;
      continue;
    }

    if (argument === '--check') {
      check = true;
      continue;
    }

    if (argument === '--output') {
      outputPathExpected = true;
      continue;
    }

    if (argument.startsWith('--')) {
      throw new Error(`未知の option です: ${argument}`);
    }

    sourcePaths.push(argument);
  }

  if (outputPathExpected) {
    throw new Error('`--output` の後に preview HTML の出力パスを指定してください。');
  }

  if (outputPath !== null && sourcePaths.length !== 1) {
    throw new Error('`--output` は JSON source を 1 つだけ指定する場合に使用できます。');
  }

  return { check, outputPath, sourcePaths };
}

/**
 * JSON source パスから対応する generated HTML preview パスを導出する。
 *
 * @param {string} sourcePath - `.wireframe.json` source のパス。
 * @returns {string} 対応する `.wireframe.html` のパス。
 * @throws {Error} source の拡張子が wireframe JSON ではない場合。
 */
function getDefaultOutputPath(sourcePath) {
  if (!sourcePath.endsWith(JSON_SUFFIX)) {
    throw new Error(`wireframe JSON ではありません: ${sourcePath}`);
  }

  return `${sourcePath.slice(0, -JSON_SUFFIX.length)}${HTML_SUFFIX}`;
}

/**
 * script 要素内へ安全に埋め込める JSON literal を生成する。
 *
 * @param {unknown} wireframe - JSON として検証済みの wireframe データ。
 * @returns {string} HTML parser による script 終端を防いだ JSON literal。
 */
function serializeForScript(wireframe) {
  return JSON.stringify(wireframe, null, 2)
    .replaceAll('<', '\\u003c')
    .replaceAll('>', '\\u003e')
    .replaceAll('&', '\\u0026')
    .replaceAll('\u2028', '\\u2028')
    .replaceAll('\u2029', '\\u2029');
}

/**
 * JSON source と preview template から、手編集を禁止した完全な HTML を生成する。
 *
 * @param {string} sourcePath - JSON source の絶対パス。
 * @param {string} outputPath - generated preview の絶対パス。
 * @returns {string} 書き込みまたは drift 比較に使う HTML 内容。
 * @throws {Error} source JSON または template が不正な場合。
 */
function renderPreview(sourcePath, outputPath) {
  const source = readFileSync(sourcePath, 'utf8');
  /** @type {unknown} */
  let wireframe;

  try {
    wireframe = JSON.parse(source);
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error);
    throw new Error(`wireframe JSON を解析できません: ${sourcePath}: ${detail}`);
  }

  if (
    wireframe === null ||
    typeof wireframe !== 'object' ||
    Array.isArray(wireframe) ||
    !('name' in wireframe) ||
    !('root' in wireframe)
  ) {
    throw new Error(`wireframe JSON には top-level の name と root が必要です: ${sourcePath}`);
  }

  const template = readFileSync(TEMPLATE_PATH, 'utf8');
  if (!template.includes(TEMPLATE_MARKER)) {
    throw new Error(`wireframe template に data marker がありません: ${TEMPLATE_PATH}`);
  }

  const sourceReference = path
    .relative(path.dirname(outputPath), sourcePath)
    .split(path.sep)
    .join('/');
  const header = `<!-- GENERATED FROM ${sourceReference} BY .opencode/skills/wireframe/scripts/generate-preview.mjs; DO NOT EDIT. -->\n`;
  const dataLiteral = serializeForScript(wireframe);
  const populatedTemplate = template.replace(
    TEMPLATE_MARKER,
    `const WIREFRAME_DATA = ${dataLiteral}; // %%WIREFRAME_DATA%%`
  );

  return `${header}${populatedTemplate}`;
}

/**
 * 指定 source を生成または検査し、失敗時に利用者が修正すべき source を返す。
 *
 * @param {string} sourcePath - JSON source のパス。
 * @param {string} outputPath - generated preview のパス。
 * @param {boolean} check - `true` の場合は書き込まず drift だけを検査する。
 * @returns {string | null} 失敗メッセージ。成功時は `null`。
 */
function generateOrCheckPreview(sourcePath, outputPath, check) {
  const absoluteSourcePath = path.resolve(sourcePath);
  const absoluteOutputPath = path.resolve(outputPath);
  let expectedPreview;

  try {
    expectedPreview = renderPreview(absoluteSourcePath, absoluteOutputPath);
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error);
    return detail;
  }

  if (!check) {
    mkdirSync(path.dirname(absoluteOutputPath), { recursive: true });
    writeFileSync(absoluteOutputPath, expectedPreview, 'utf8');
    return null;
  }

  let actualPreview;
  try {
    actualPreview = readFileSync(absoluteOutputPath, 'utf8');
  } catch {
    return `generated preview がありません: ${absoluteOutputPath}. JSON を編集して generator を実行してください。`;
  }

  if (actualPreview !== expectedPreview) {
    return `generated preview が JSON source と一致しません: ${absoluteOutputPath}. HTML を編集せず ${absoluteSourcePath} から再生成してください。`;
  }

  return null;
}

let options;
try {
  options = parseArguments(process.argv.slice(2));
} catch (error) {
  const detail = error instanceof Error ? error.message : String(error);
  process.stderr.write(`Wireframe preview generator failed: ${detail}\n`);
  process.exitCode = 1;
  options = null;
}

if (options !== null) {
  if (options.sourcePaths.length === 0) {
    process.stderr.write(
      'Usage: node .opencode/skills/wireframe/scripts/generate-preview.mjs [--check] <source.wireframe.json> [--output <preview.wireframe.html>]\n'
    );
    process.exitCode = 1;
  } else {
    const errors = [];
    for (const sourcePath of options.sourcePaths) {
      let outputPath;
      try {
        outputPath = options.outputPath ?? getDefaultOutputPath(sourcePath);
      } catch (error) {
        const detail = error instanceof Error ? error.message : String(error);
        errors.push(detail);
        continue;
      }
      const error = generateOrCheckPreview(sourcePath, outputPath, options.check);
      if (error !== null) errors.push(error);
    }

    if (errors.length > 0) {
      process.stderr.write(
        `Wireframe preview generator failed:\n${errors.map((error) => `- ${error}`).join('\n')}\n`
      );
      process.exitCode = 1;
    }
  }
}

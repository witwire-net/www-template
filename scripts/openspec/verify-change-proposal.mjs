import { existsSync, readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { collectActiveChangeDirectories } from '#openspec/change-artifacts.mjs';

const PROPOSAL_FILE_NAME = 'proposal.md';
const RESOLUTIONS = new Set(['DRAFT', 'REQUEST_SUFFICIENT', 'OWNER_CONFIRMED']);
const RESOLVED = new Set(['REQUEST_SUFFICIENT', 'OWNER_CONFIRMED']);
const UX_MODES = new Set(['NONE', 'CONTINUITY', 'SHAPE']);
const CLASSIFICATIONS = new Set([
  'Desired Outcome',
  'Outcome Constraint',
  'Required Means',
  'Candidate Means',
]);
const REQUIRED_HEADINGS = [
  'Outcome',
  'Why',
  'Scope',
  'Request Classification',
  'Spec Units',
  'UI / UX Impact',
  'Material Constraints',
  'Repository Evidence',
  'Assumptions and Decisions',
  'Observable Success',
  'Confirmation Evidence',
];
const REQUIRED_SUBHEADINGS = ['In Scope', 'Out of Scope', 'New Spec Units', 'Modified Spec Units'];
const PLACEHOLDER_PATTERN = /<!--\s*TODO:|\bTBD\b/iu;

/**
 * HTML コメントを同じ改行数の空白へ置換し、コメント内の例示見出しを文書構造と誤認しないようにする。
 *
 * @param {string} source - proposal.md の完全な内容。
 * @returns {string} 行番号を保持した検査用内容。
 */
function removeHtmlComments(source) {
  return source.replace(/<!--[\s\S]*?-->/gu, (comment) =>
    '\n'.repeat(comment.split(/\r?\n/u).length - 1)
  );
}

/**
 * `Key: VALUE` 形式の提案マーカーを一件だけ取得する。
 *
 * @param {string} source - コメントを除いた proposal.md の内容。
 * @param {string} key - 取得するマーカー名。
 * @returns {string | null} マーカー値。欠落または重複時は `null`。
 */
function getUniqueMarker(source, key) {
  const matches = [...source.matchAll(/^(Intent-Resolution|UX-Mode):\s*([A-Z_]+)\s*$/gmu)].filter(
    (match) => match[1] === key
  );
  return matches.length === 1 ? (matches[0]?.[2] ?? null) : null;
}

/**
 * 指定階層の Markdown 見出しを名前、行番号、文字位置として収集する。
 *
 * @param {string} source - コメントを除いた proposal.md の内容。
 * @param {2 | 3} level - 収集する見出し階層。
 * @returns {{ name: string; line: number; index: number }[]} 文書順の見出し一覧。
 */
function collectHeadings(source, level) {
  const headings = [];
  for (const match of source.matchAll(/^(#{2,3})\s+(.+?)\s*$/gmu)) {
    if (match[1]?.length !== level) continue;
    const index = match.index ?? 0;
    headings.push({
      name: match[2]?.trim() ?? '',
      line: source.slice(0, index).split(/\r?\n/u).length,
      index,
    });
  }
  return headings;
}

/**
 * 指定見出しから次の同階層以上の見出しまでを節本文として取得する。
 *
 * @param {string} source - コメントを除いた proposal.md の内容。
 * @param {{ index: number }} heading - 本文を取得する見出し位置。
 * @param {number} level - 対象見出しの階層。
 * @returns {string} 見出し行を除いた節本文。
 */
function getSectionBody(source, heading, level) {
  const headingLineEnd = source.indexOf('\n', heading.index);
  const bodyStart = headingLineEnd < 0 ? source.length : headingLineEnd + 1;
  const nextHeadingPattern = /^(#{1,3})\s+/gmu;
  nextHeadingPattern.lastIndex = bodyStart;
  let nextHeading = nextHeadingPattern.exec(source);
  while (nextHeading?.[1] !== undefined && nextHeading[1].length > level) {
    nextHeading = nextHeadingPattern.exec(source);
  }
  return source.slice(bodyStart, nextHeading?.index ?? source.length);
}

/**
 * 節本文に見出し記号や表罫線以外の説明があるかを判定する。
 *
 * @param {string} body - 検査対象の節本文。
 * @returns {boolean} 文字または数字を含む説明がある場合は `true`。
 */
function hasMeaningfulContent(body) {
  return /[\p{L}\p{N}]/u.test(body.replace(/^#{1,6}\s+.*$/gmu, ''));
}

/**
 * Change 配下に proposal より後段の成果物があるかを再帰的に確認する。
 *
 * @param {string} changeDirectory - 活動中 Change ディレクトリの絶対パス。
 * @returns {string[]} proposal.md と管理ファイルを除く後続成果物の絶対パス。
 */
function collectDownstreamArtifacts(changeDirectory) {
  const artifacts = [];
  const pending = [changeDirectory];

  while (pending.length > 0) {
    const current = pending.pop();
    if (!current) continue;

    for (const entry of readdirSync(current, { withFileTypes: true })) {
      const entryPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        pending.push(entryPath);
      } else if (
        entry.isFile() &&
        entry.name !== PROPOSAL_FILE_NAME &&
        entry.name !== '.openspec.yaml'
      ) {
        artifacts.push(entryPath);
      }
    }
  }

  return artifacts.sort();
}

/**
 * Request Classification 表のデータ行から分類値を抽出する。
 *
 * @param {string} source - コメントを除いた proposal.md の内容。
 * @returns {string[]} 表へ記載された分類値。
 */
function collectClassifications(source) {
  const sectionMatch = /^## Request Classification\s*$([\s\S]*?)(?=^##\s+|(?![\s\S]))/mu.exec(
    source
  );
  if (!sectionMatch?.[1]) return [];

  const classifications = [];
  for (const line of sectionMatch[1].split(/\r?\n/u)) {
    const cells = line.split('|').map((cell) => cell.trim());
    if (cells.length < 5 || cells[2] === 'Classification' || /^-+$/u.test(cells[2] ?? '')) continue;
    if (cells[2]) classifications.push(cells[2]);
  }
  return classifications;
}

/**
 * 検査失敗へリポジトリ相対パスと行番号を付与する。
 *
 * @param {string[]} errors - エラーを蓄積する配列。
 * @param {string} absolutePath - 問題を検出した成果物の絶対パス。
 * @param {number} line - 1 始まりの行番号。
 * @param {string} message - 利用者が修正判断できる説明。
 * @returns {void}
 */
function addError(errors, absolutePath, line, message) {
  errors.push(`${path.relative(process.cwd(), absolutePath)}:${String(line)}: ${message}`);
}

const errors = [];

for (const changeDirectory of collectActiveChangeDirectories(process.cwd())) {
  const proposalPath = path.join(changeDirectory, PROPOSAL_FILE_NAME);
  const downstreamArtifacts = collectDownstreamArtifacts(changeDirectory);

  // 空の Change は準備状態として許可するが、後続成果物だけがある不完全な状態は拒否する。
  if (!existsSync(proposalPath)) {
    if (downstreamArtifacts.length > 0) {
      addError(errors, changeDirectory, 1, '後続成果物を作成する前に proposal.md が必要です。');
    }
    continue;
  }

  const originalSource = readFileSync(proposalPath, 'utf8');
  const source = removeHtmlComments(originalSource);
  const resolution = getUniqueMarker(source, 'Intent-Resolution');
  const uxMode = getUniqueMarker(source, 'UX-Mode');

  if (resolution === null || !RESOLUTIONS.has(resolution)) {
    addError(
      errors,
      proposalPath,
      1,
      'Intent-Resolution は DRAFT、REQUEST_SUFFICIENT、OWNER_CONFIRMED のいずれか一つでなければなりません。'
    );
  }
  if (uxMode === null || !UX_MODES.has(uxMode)) {
    addError(
      errors,
      proposalPath,
      2,
      'UX-Mode は NONE、CONTINUITY、SHAPE のいずれか一つでなければなりません。'
    );
  }

  // DRAFT は提案内容の作業途中を許可する一方、仕様以降へ進むことは許可しない。
  if (resolution === 'DRAFT') {
    if (downstreamArtifacts.length > 0) {
      addError(
        errors,
        proposalPath,
        1,
        'Intent-Resolution が DRAFT の間は後続成果物を作成できません。'
      );
    }
    continue;
  }
  if (!RESOLVED.has(resolution ?? '')) continue;

  const h2Headings = collectHeadings(source, 2);
  const h2Names = h2Headings.map(({ name }) => name).filter((name) => name !== 'Thesaurus');
  if (h2Names.join('\u0000') !== REQUIRED_HEADINGS.join('\u0000')) {
    addError(
      errors,
      proposalPath,
      1,
      `解決済み提案の見出しは指定順で ${REQUIRED_HEADINGS.map((heading) => `## ${heading}`).join('、')} としなければなりません。`
    );
  }

  for (const heading of h2Headings) {
    if (!hasMeaningfulContent(getSectionBody(source, heading, 2))) {
      addError(errors, proposalPath, heading.line, `## ${heading.name} に内容がありません。`);
    }
  }

  const h3Headings = collectHeadings(source, 3);
  const h3Names = new Set(h3Headings.map(({ name }) => name));
  for (const heading of REQUIRED_SUBHEADINGS) {
    const entry = h3Headings.find(({ name }) => name === heading);
    if (!entry) {
      addError(errors, proposalPath, 1, `必須見出し '### ${heading}' がありません。`);
    } else if (!hasMeaningfulContent(getSectionBody(source, entry, 3))) {
      addError(errors, proposalPath, entry.line, `### ${heading} に内容がありません。`);
    }
  }

  const placeholderMatch = PLACEHOLDER_PATTERN.exec(originalSource);
  if (placeholderMatch?.index !== undefined) {
    const line = originalSource.slice(0, placeholderMatch.index).split(/\r?\n/u).length;
    addError(errors, proposalPath, line, '解決済み提案に TODO または TBD を残せません。');
  }

  const classifications = collectClassifications(source);
  if (classifications.length === 0) {
    addError(errors, proposalPath, 1, 'Request Classification に少なくとも一つの分類が必要です。');
  }
  for (const classification of classifications) {
    if (!CLASSIFICATIONS.has(classification)) {
      addError(
        errors,
        proposalPath,
        1,
        `Request Classification の分類 '${classification}' は許可されていません。`
      );
    }
  }

  // UX モード固有の証跡が揃っていることを確認し、UI の意図が暗黙に決まる状態を防ぐ。
  if (uxMode === 'CONTINUITY' && !h3Names.has('Continuity Source')) {
    addError(
      errors,
      proposalPath,
      1,
      'UX-Mode が CONTINUITY の場合は ### Continuity Source が必要です。'
    );
  }
  if (uxMode === 'SHAPE') {
    for (const heading of ['Primary User Task', 'UX Direction']) {
      if (!h3Names.has(heading))
        addError(errors, proposalPath, 1, `UX-Mode が SHAPE の場合は ### ${heading} が必要です。`);
    }
  }
  const uxSection = h2Headings.find(({ name }) => name === 'UI / UX Impact');
  const nextH2 = h2Headings.find(({ index }) => uxSection && index > uxSection.index);
  for (const heading of ['Continuity Source', 'Primary User Task', 'UX Direction']) {
    const entry = h3Headings.find(({ name }) => name === heading);
    if (
      entry &&
      uxSection &&
      (entry.index < uxSection.index || entry.index >= (nextH2?.index ?? source.length))
    ) {
      addError(
        errors,
        proposalPath,
        entry.line,
        `### ${heading} は ## UI / UX Impact 内に配置してください。`
      );
    }
  }
}

if (errors.length > 0) {
  process.stderr.write(
    `OpenSpec Change proposal guard failed:\n${errors.map((error) => `- ${error}`).join('\n')}\n`
  );
  process.exitCode = 1;
}

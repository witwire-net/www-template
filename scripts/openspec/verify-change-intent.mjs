import { existsSync, readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { isDirectory } from '#openspec/change-artifacts.mjs';

// intent の状態値と downstream artifact の存在を別々に検査し、作業途中の DRAFT 自体は許可します。
const INTENT_FILE_NAME = 'intent.md';
const CONFIRMED_STATUS = 'CONFIRMED';
const ALLOWED_INTENT_STATUSES = new Set(['DRAFT', CONFIRMED_STATUS]);
const ALLOWED_CONFIRMATION_STATUSES = new Set(['PENDING', CONFIRMED_STATUS]);
const ROOT_DOWNSTREAM_FILE_NAMES = new Set(['proposal.md', 'design.md', 'tasks.md']);
const PLACEHOLDER_PATTERN = /<!--\s*TODO:|\bTBD\b/iu;
const REQUIRED_HEADINGS = [
  'Customer / Owner Outcome',
  'Request Classification',
  'Repository Evidence',
  'Inferences and Assumptions',
  'Falsification Check',
  'Invariants and Boundaries',
  'Observable Success',
  'Owner Confirmation',
];

/**
 * 未アーカイブ Change の直下ディレクトリを列挙する。
 *
 * OpenSpec の archive は確定済み履歴なので、現在の Intent 確認ゲートから除外します。
 *
 * @param {string} repositoryRoot - リポジトリ root の絶対パス。
 * @returns {string[]} 検査対象となる Change ディレクトリの絶対パス。
 */
function collectActiveChangeDirectories(repositoryRoot) {
  const changesDirectory = path.join(repositoryRoot, 'openspec', 'changes');
  if (!isDirectory(changesDirectory)) return [];

  return readdirSync(changesDirectory, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && entry.name !== 'archive')
    .map((entry) => path.join(changesDirectory, entry.name));
}

/**
 * Intent 確認後にのみ作成できる downstream artifact を再帰的に収集する。
 *
 * OpenSpec 管理ファイルと intent.md は除外し、proposal、spec、design、tasks、wireframe JSON、
 * screenshot を確認ゲートの対象にします。
 *
 * @param {string} changeDirectory - 対象 Change ディレクトリの絶対パス。
 * @returns {string[]} downstream artifact の絶対パス。
 */
function collectDownstreamArtifacts(changeDirectory) {
  const artifacts = [];
  const pendingDirectories = [changeDirectory];

  while (pendingDirectories.length > 0) {
    const currentDirectory = pendingDirectories.pop();
    if (!currentDirectory) continue;

    for (const entry of readdirSync(currentDirectory, { withFileTypes: true })) {
      const entryPath = path.join(currentDirectory, entry.name);
      if (entry.isDirectory()) {
        pendingDirectories.push(entryPath);
        continue;
      }

      if (!entry.isFile() || entry.name === INTENT_FILE_NAME || entry.name === '.openspec.yaml') {
        continue;
      }

      const isRootArtifact =
        currentDirectory === changeDirectory && ROOT_DOWNSTREAM_FILE_NAMES.has(entry.name);
      const isDeltaSpec =
        entry.name === 'spec.md' && entryPath.includes(`${path.sep}specs${path.sep}`);
      const isWireframeSource = entry.name.endsWith('.wireframe.json');
      const isWireframeScreenshot = entry.name.endsWith('.wireframe-screenshot.png');

      if (isRootArtifact || isDeltaSpec || isWireframeSource || isWireframeScreenshot) {
        artifacts.push(entryPath);
      }
    }
  }

  return artifacts;
}

/**
 * `Key: VALUE` 形式の status marker を artifact から取得する。
 *
 * @param {string} source - intent.md の完全な内容。
 * @param {string} key - 取得する marker 名。
 * @returns {string | null} marker 値。存在しない場合は `null`。
 */
function getStatusMarker(source, key) {
  const pattern =
    key === 'Intent-Status'
      ? /^Intent-Status:\s*([A-Z]+)\s*$/mu
      : /^Owner-Confirmation:\s*([A-Z]+)\s*$/mu;
  const match = pattern.exec(source);
  return match?.[1] ?? null;
}

/**
 * 指定した文字位置を 1 始まりの行番号へ変換する。
 *
 * @param {string} source - 行番号を計算する完全な内容。
 * @param {number} index - 0 始まりの文字位置。
 * @returns {number} 1 始まりの行番号。
 */
function getLineNumber(source, index) {
  return source.slice(0, index).split(/\r?\n/u).length;
}

/**
 * 検査エラーへリポジトリ相対パスと行番号を付与する。
 *
 * @param {string[]} errors - エラーを蓄積する配列。
 * @param {string} absolutePath - 問題を検出したファイルまたはディレクトリ。
 * @param {number} line - 1 始まりの行番号。
 * @param {string} message - 利用者向けの日本語エラー説明。
 */
function addError(errors, absolutePath, line, message) {
  errors.push(`${path.relative(process.cwd(), absolutePath)}:${line}: ${message}`);
}

const errors = [];

for (const changeDirectory of collectActiveChangeDirectories(process.cwd())) {
  const intentPath = path.join(changeDirectory, INTENT_FILE_NAME);
  const downstreamArtifacts = collectDownstreamArtifacts(changeDirectory);

  // downstream artifact がなければ、Intent 候補をまだファイル化していない初期状態を許可します。
  if (!existsSync(intentPath)) {
    if (downstreamArtifacts.length > 0) {
      addError(
        errors,
        changeDirectory,
        1,
        `確認済み intent.md がない状態で downstream artifact '${path.relative(changeDirectory, downstreamArtifacts[0])}' は作成できません。`
      );
    }
    continue;
  }

  const source = readFileSync(intentPath, 'utf8');
  const intentStatus = getStatusMarker(source, 'Intent-Status');
  const confirmationStatus = getStatusMarker(source, 'Owner-Confirmation');

  if (intentStatus === null || !ALLOWED_INTENT_STATUSES.has(intentStatus)) {
    addError(
      errors,
      intentPath,
      1,
      'Intent-Status は DRAFT または CONFIRMED でなければなりません。'
    );
  }
  if (confirmationStatus === null || !ALLOWED_CONFIRMATION_STATUSES.has(confirmationStatus)) {
    addError(
      errors,
      intentPath,
      2,
      'Owner-Confirmation は PENDING または CONFIRMED でなければなりません。'
    );
  }
  const intentClaimsConfirmation = intentStatus === CONFIRMED_STATUS;
  const ownerClaimsConfirmation = confirmationStatus === CONFIRMED_STATUS;
  if (intentClaimsConfirmation !== ownerClaimsConfirmation) {
    addError(
      errors,
      intentPath,
      1,
      'Intent-Status と Owner-Confirmation は DRAFT/PENDING または CONFIRMED/CONFIRMED の組み合わせでなければなりません。'
    );
  }

  const isConfirmed = intentClaimsConfirmation && ownerClaimsConfirmation;
  if (downstreamArtifacts.length > 0 && !isConfirmed) {
    addError(
      errors,
      intentPath,
      1,
      `Intent が CONFIRMED になる前に downstream artifact '${path.relative(changeDirectory, downstreamArtifacts[0])}' は作成できません。`
    );
  }

  // 確認済みと宣言した Intent は、構造と placeholder の両方を検査して空の承認を拒否します。
  if (!isConfirmed) continue;

  const sourceLines = new Set(source.split(/\r?\n/u));
  for (const heading of REQUIRED_HEADINGS) {
    if (!sourceLines.has(`## ${heading}`)) {
      addError(
        errors,
        intentPath,
        1,
        `確認済み Intent に必須見出し '## ${heading}' がありません。`
      );
    }
  }

  const placeholderMatch = PLACEHOLDER_PATTERN.exec(source);
  if (placeholderMatch?.index !== undefined) {
    addError(
      errors,
      intentPath,
      getLineNumber(source, placeholderMatch.index),
      '確認済み Intent に TODO または TBD placeholder を残せません。'
    );
  }
}

if (errors.length > 0) {
  process.stderr.write(
    `OpenSpec Change intent guard failed:\n${errors.map((error) => `- ${error}`).join('\n')}\n`
  );
  process.exitCode = 1;
}

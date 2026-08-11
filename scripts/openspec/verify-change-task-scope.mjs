import { readFileSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { collectActiveChangeArtifacts } from '#openspec/change-artifacts.mjs';

const TASK_FILE_NAME = 'tasks.md';
const DESIGN_FILE_NAME = 'design.md';
const CHECKBOX_PATTERN = /^- \[[ xX]\]\s+/u;
const WORK_PACKAGE_PATTERN = /^- \[[ xX]\] WP\d+:\s+\S/u;
const REQUIRED_WORK_PACKAGE_FIELDS = [
  { name: 'Covers', pattern: /^\s+-\s+Covers:\s+\S/mu },
  { name: 'Completion Evidence', pattern: /^\s+-\s+Completion Evidence:\s+\S/mu },
];
const ARCHITECTURE_DESIGN_HEADINGS = [
  'Material Decisions',
  'Boundaries',
  'Data',
  'Security',
  'Migration',
  'Rollback',
  'Failure Modes',
  'Risks',
  'Verification',
  'Revisit Triggers',
];
const FILE_PLAN_PATTERN =
  /(?:^|[\s`(])(?:[\w@.-]+\/)+[\w.-]+|\b[\w.-]+\.(?:cjs|css|go|html|js|jsx|json|md|mjs|sql|svelte|ts|tsx|yaml|yml)\b/iu;
const HELPER_PLAN_PATTERN =
  /\b(?:class|component|function|helper|hook|method|utility)\b|(?:クラス|コンポーネント|関数|補助処理|ヘルパー|フック|メソッド)(?:を|の|へ|ごと)/iu;
const TEST_LAYER_PLAN_PATTERN =
  /\b(?:component|e2e|integration|unit) tests?\b|(?:単体|結合|統合|コンポーネント|E2E)試験(?:を|の|へ|ごと)/iu;
const EXTERNAL_OPERATION_PATTERNS = [
  {
    label: 'リリースまたはデプロイの実行',
    pattern: /(?:リリース|デプロイ)(?:を|の)?(?:実行|実施)|\bwrangler\s+deploy\b/iu,
  },
  {
    label: '外部環境の操作または検証',
    pattern:
      /(?:本番|production|staging|ステージング).{0,32}(?:作成|接続|確認|検証|監視|操作|テスト)/iu,
  },
  {
    label: '認証情報へのアクセス',
    pattern:
      /(?:credential|credentials|認証情報|アクセストークン|API token|secret).{0,32}(?:取得|入力|要求|確認|検証)/iu,
  },
  {
    label: '外部承認',
    pattern: /(?:外部|運用担当|operator|第三者).{0,24}(?:承認|approval|許可|依頼)/iu,
  },
];

/**
 * @typedef {{ line: number; summary: string; body: string }} WorkPackage
 */

/**
 * tasks.md のチェック項目と、その項目に属する説明行を一つの作業パッケージ候補へまとめる。
 *
 * @param {string} source - tasks.md の完全な内容。
 * @returns {WorkPackage[]} チェック項目の開始行、要約、本文。
 */
function collectWorkPackages(source) {
  const lines = source.split(/\r?\n/u);
  const packages = [];
  let current = null;

  for (const [index, line] of lines.entries()) {
    if (CHECKBOX_PATTERN.test(line)) {
      if (current) packages.push(current);
      current = { line: index + 1, summary: line, body: line };
      continue;
    }
    if (/^##\s+/u.test(line)) {
      if (current) packages.push(current);
      current = null;
      continue;
    }
    if (current) current.body += `\n${line}`;
  }
  if (current) packages.push(current);
  return packages;
}

/**
 * 作業パッケージの要約と Covers だけを粒度検査用の計画文として取得する。
 *
 * Completion Evidence に現れるコマンドや試験種別は実装分解ではなく証跡なので除外する。
 *
 * @param {WorkPackage} workPackage - 検査する作業パッケージ。
 * @returns {string} 粒度を判断する計画部分。
 */
function getPlanningText(workPackage) {
  const lines = workPackage.body.split(/\r?\n/u);
  return lines.filter((line) => !/^\s*-\s+Completion Evidence:/u.test(line)).join('\n');
}

/**
 * 禁止された外部操作の種別を文章から一件取得する。
 *
 * @param {string} source - 作業パッケージまたは設計の内容。
 * @returns {string | null} 該当する禁止操作。該当しない場合は `null`。
 */
function getExternalOperation(source) {
  return EXTERNAL_OPERATION_PATTERNS.find(({ pattern }) => pattern.test(source))?.label ?? null;
}

/**
 * Markdown の第2階層見出しを文書順に収集する。
 *
 * @param {string} source - design.md の完全な内容。
 * @returns {{ name: string; line: number }[]} 見出し名と 1 始まり行番号。
 */
function collectLevelTwoHeadings(source) {
  const headings = [];
  for (const [index, line] of source.split(/\r?\n/u).entries()) {
    const name = /^##\s+(.+?)\s*$/u.exec(line)?.[1];
    if (name) headings.push({ name, line: index + 1 });
  }
  return headings;
}

/**
 * 診断へリポジトリ相対パスと行番号を付与する。
 *
 * @param {string[]} errors - 診断を蓄積する配列。
 * @param {string} absolutePath - 問題がある成果物の絶対パス。
 * @param {number} line - 1 始まりの行番号。
 * @param {string} message - 修正方法が判断できる説明。
 * @returns {void}
 */
function addError(errors, absolutePath, line, message) {
  errors.push(`${path.relative(process.cwd(), absolutePath)}:${String(line)}: ${message}`);
}

const errors = [];

for (const taskPath of collectActiveChangeArtifacts(
  process.cwd(),
  (_absolutePath, fileName) => fileName === TASK_FILE_NAME
)) {
  const source = readFileSync(taskPath, 'utf8');
  const workPackages = collectWorkPackages(source);
  if (workPackages.length === 0)
    addError(errors, taskPath, 1, 'tasks.md には少なくとも一つの Work Package が必要です。');

  for (const workPackage of workPackages) {
    if (!WORK_PACKAGE_PATTERN.test(workPackage.summary)) {
      addError(
        errors,
        taskPath,
        workPackage.line,
        'チェック項目は `- [ ] WP<number>: <成果>` 形式で記載してください。'
      );
    }
    for (const field of REQUIRED_WORK_PACKAGE_FIELDS) {
      if (!field.pattern.test(workPackage.body)) {
        addError(
          errors,
          taskPath,
          workPackage.line,
          `Work Package には内容を持つ ${field.name}: が必要です。`
        );
      }
    }

    const planningText = getPlanningText(workPackage);
    if (FILE_PLAN_PATTERN.test(planningText)) {
      addError(
        errors,
        taskPath,
        workPackage.line,
        'Work Package をファイル単位の計画へ分解できません。成果のまとまりで記載してください。'
      );
    }
    if (HELPER_PLAN_PATTERN.test(planningText)) {
      addError(
        errors,
        taskPath,
        workPackage.line,
        'Work Package を補助処理またはコード要素単位の計画へ分解できません。'
      );
    }
    if (TEST_LAYER_PLAN_PATTERN.test(planningText)) {
      addError(
        errors,
        taskPath,
        workPackage.line,
        'Work Package を試験階層単位の計画へ分解できません。検証は Completion Evidence に集約してください。'
      );
    }
    const externalOperation = getExternalOperation(workPackage.body);
    if (externalOperation) {
      addError(
        errors,
        taskPath,
        workPackage.line,
        `${externalOperation} は Work Package または完了条件に含められません。`
      );
    }
  }
}

for (const designPath of collectActiveChangeArtifacts(
  process.cwd(),
  (_absolutePath, fileName) => fileName === DESIGN_FILE_NAME
)) {
  const source = readFileSync(designPath, 'utf8');
  const headings = collectLevelTwoHeadings(source);
  const names = headings.map(({ name }) => name);

  // architecture-change の設計を物質的判断だけに限定し、旧来のファイル一覧や詳細計画を再導入させない。
  if (names.join('\u0000') !== ARCHITECTURE_DESIGN_HEADINGS.join('\u0000')) {
    addError(
      errors,
      designPath,
      1,
      `design.md の見出しは指定順で ${ARCHITECTURE_DESIGN_HEADINGS.map((heading) => `## ${heading}`).join('、')} だけを使用してください。`
    );
  }
  const externalOperation = getExternalOperation(source);
  if (externalOperation)
    addError(errors, designPath, 1, `${externalOperation} は設計の完了条件に含められません。`);
}

if (errors.length > 0) {
  process.stderr.write(
    `OpenSpec Change task scope guard failed:\n${errors.map((error) => `- ${error}`).join('\n')}\n`
  );
  process.exitCode = 1;
}

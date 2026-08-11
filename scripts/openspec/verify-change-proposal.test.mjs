import assert from 'node:assert/strict';
import test from 'node:test';
import { fileURLToPath, URL } from 'node:url';

import { runGuardInFixture } from '#openspec/guard-test-fixture.mjs';

const guardScriptPath = fileURLToPath(new URL('./verify-change-proposal.mjs', import.meta.url));

/**
 * 条件分岐の検査に使う、解決済み提案の完全な基準内容を生成する。
 *
 * @param {{ resolution?: string; uxMode?: string; uxDetails?: string }} [options] - 差し替える意図解決値と UX 証跡。
 * @returns {string} proposal.md として有効な内容。
 */
function createResolvedProposal({
  resolution = 'REQUEST_SUFFICIENT',
  uxMode = 'NONE',
  uxDetails = '利用者に見える画面または操作は変化しない。',
} = {}) {
  return `Intent-Resolution: ${resolution}
UX-Mode: ${uxMode}

## Outcome

利用者が一貫した仕様に基づく成果を得られる。

## Why

活動中の仕様差分を含めなければ、実装前の不整合を検出できないため。

## Scope

### In Scope

- 活動中の仕様差分を検査対象に含める。

### Out of Scope

- 外部環境での運用は対象外とする。

## Request Classification

| Request Statement | Classification | Resolution |
| --- | --- | --- |
| 仕様差分を検査する | Desired Outcome | 活動中の差分を含む実効仕様を検査する。 |
| 標準ライブラリを使う | Required Means | 実装上の制約として扱う。 |

## Spec Units

### New Spec Units

- \`spec-governance\`: 仕様検査の契約を扱う。

### Modified Spec Units

- なし。既存の仕様単位は変更しない。

## UI / UX Impact

${uxDetails}

## Material Constraints

- 外部依存を追加しない。

## Repository Evidence

| Source | Observation | Relevance |
| --- | --- | --- |
| \`scripts/openspec\` | 検査処理が存在する。 | 共通処理を再利用できる。 |

## Assumptions and Decisions

- Assumptions: なし。対象範囲は依頼で明示されている。
- Decisions: 依頼で指定された成果を採用する。

## Observable Success

- 活動中の差分にあるシナリオが検査される。

## Confirmation Evidence

- 依頼文が成果、範囲、制約を明示している。
`;
}

test('REQUEST_SUFFICIENT の完全な提案を許可する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': createResolvedProposal(),
  });

  assert.equal(result.status, 0);
  assert.equal(result.stderr, '');
});

test('後続成果物のない DRAFT 提案を許可する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md':
      'Intent-Resolution: DRAFT\nUX-Mode: NONE\n\n<!-- TODO: 意図を確認する。 -->\n',
  });

  assert.equal(result.status, 0);
});

test('proposal.md がない後続成果物を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/specs/account/spec.md': '## Purpose\n\nアカウントを扱う。\n',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /proposal\.md が必要/u);
});

test('DRAFT 提案の後続成果物を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': 'Intent-Resolution: DRAFT\nUX-Mode: NONE\n',
    'openspec/changes/example/tasks.md': '## Work Packages\n',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /DRAFT の間/u);
});

test('解決済み提案に残る placeholder を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': `${createResolvedProposal()}\n<!-- TODO: 未解決 -->\n`,
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /TODO または TBD/u);
});

test('解決済み提案の空の確認証跡を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': createResolvedProposal().replace(
      '- 依頼文が成果、範囲、制約を明示している。',
      '-'
    ),
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Confirmation Evidence に内容がありません/u);
});

test('指定外の分類を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': createResolvedProposal().replace(
      'Desired Outcome',
      'Implementation'
    ),
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Implementation.*許可されていません/u);
});

test('CONTINUITY に既存体験の根拠を要求する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': createResolvedProposal({ uxMode: 'CONTINUITY' }),
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Continuity Source/u);
});

test('SHAPE に中心作業と体験方向を要求する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': createResolvedProposal({
      resolution: 'OWNER_CONFIRMED',
      uxMode: 'SHAPE',
      uxDetails: '### Primary User Task\n\n利用者が仕様を確認する。',
    }),
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /UX Direction/u);
});

test('UX 固有見出しを UI / UX Impact の外へ置くことを拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': createResolvedProposal({
      uxMode: 'CONTINUITY',
      uxDetails: '既存の画面構成を維持する。',
    }).replace(
      '## Material Constraints',
      '## Material Constraints\n\n### Continuity Source\n\n`packages/frontend` の既存画面。'
    ),
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /UI \/ UX Impact 内/u);
});

test('archive 配下の履歴は検査しない', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/archive/example/tasks.md': '不完全な履歴',
  });

  assert.equal(result.status, 0);
});

import assert from 'node:assert/strict';
import test from 'node:test';
import { fileURLToPath, URL } from 'node:url';

import { runGuardInFixture } from '#openspec/guard-test-fixture.mjs';

const guardScriptPath = fileURLToPath(new URL('./verify-change-intent.mjs', import.meta.url));

const confirmedIntent = `Intent-Status: CONFIRMED
Owner-Confirmation: CONFIRMED

## Customer / Owner Outcome

- Actor: 開発者
- Situation: OpenSpec Change を作成するとき
- Problem: 解法が意図より先に固定される
- Desired Outcome: 確認された目的から仕様を作成できる
- Priority: 顧客成果を優先する

## Request Classification

| Request Term / Statement | Classification | Confirmed Meaning |
| --- | --- | --- |
| 認証を実装する | Candidate Means | 安全に本人確認できる成果を求めている |

## Repository Evidence

| Evidence Type | Source | Observation | Interpretation |
| --- | --- | --- | --- |
| Observed Fact | \`AGENTS.md:5\` | 根拠確認が必要 | 意図にも根拠が必要 |

## Inferences and Assumptions

- Inferences: 意図確認を先に行う
- Assumptions: なし。所有者が確認した
- Unresolved Decisions: なし。意図を変える判断は解決済み

## Falsification Check

- Materially Different Interpretation: 実装方式自体が必須制約である可能性
- Evidence Checked: 所有者へ分類を確認した
- Conclusion: 実装方式は候補手段として確認された

## Invariants and Boundaries

- Invariants: 確認済み意図を保持する
- Boundaries: repository 内の Change 作成を対象とする

## Observable Success

- proposal が確認済み意図に追跡できる

## Owner Confirmation

- Confirmed Intent: 確認された顧客成果から仕様を作成する
- Confirmation Evidence: 所有者が「この意図で確定する」と回答した
`;

test('downstream artifact と完全な確認済み Intent を許可する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/intent.md': confirmedIntent,
    'openspec/changes/example/proposal.md': '## Why\n\n確認済み意図から作成する。\n',
  });

  assert.equal(result.status, 0);
  assert.equal(result.stderr, '');
});

test('downstream artifact のない DRAFT Intent を許可する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/intent.md':
      'Intent-Status: DRAFT\nOwner-Confirmation: PENDING\n\n<!-- TODO: 意図候補 -->\n',
  });

  assert.equal(result.status, 0);
  assert.equal(result.stderr, '');
});

test('Intent がない downstream artifact を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/proposal.md': '## Why\n',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /確認済み intent\.md がない/u);
});

test('DRAFT Intent より後の downstream artifact を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/intent.md': 'Intent-Status: DRAFT\nOwner-Confirmation: PENDING\n',
    'openspec/changes/example/specs/account/spec.md': '## ADDED Requirements\n',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Intent が CONFIRMED になる前/u);
});

test('Intent と所有者確認の不一致を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/intent.md': 'Intent-Status: CONFIRMED\nOwner-Confirmation: PENDING\n',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /DRAFT\/PENDING または CONFIRMED\/CONFIRMED/u);
});

test('placeholder が残る確認済み Intent を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/intent.md': `${confirmedIntent}\n<!-- TODO: 未入力 -->\n`,
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /placeholder/u);
});

test('必須見出しが欠ける確認済み Intent を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/example/intent.md': confirmedIntent.replace(
      '## Falsification Check',
      '## 確認'
    ),
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Falsification Check/u);
});

test('archive 配下の履歴は検査しない', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/archive/example/proposal.md': '## Why\n',
  });

  assert.equal(result.status, 0);
  assert.equal(result.stderr, '');
});

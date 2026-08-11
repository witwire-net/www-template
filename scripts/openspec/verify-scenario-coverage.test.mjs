import assert from 'node:assert/strict';
import test from 'node:test';
import { fileURLToPath, URL } from 'node:url';

import { runGuardInFixture } from '#openspec/guard-test-fixture.mjs';

const guardScriptPath = fileURLToPath(new URL('./verify-scenario-coverage.mjs', import.meta.url));

/**
 * 主仕様の一要件を作成し、差分適用試験の共通基準として使用する。
 *
 * @param {string} scenarioId - 主仕様へ記載する Scenario ID。
 * @returns {string} OpenSpec 主仕様の内容。
 */
function createMainSpec(scenarioId = 'ACCOUNT-S001') {
  return `## Purpose

利用者のアカウント操作を保証する。

## Requirements

### Requirement: アカウント表示

システムはアカウントを表示しなければならない。

#### Scenario: アカウントを表示する (${scenarioId})

- **WHEN** 利用者がアカウントを開く
- **THEN** アカウントが表示される
`;
}

/**
 * 一要件を含む差分仕様を作成する。
 *
 * @param {{ kind?: string; requirement?: string; scenarioId?: string; manual?: boolean }} [options] - 差分操作と Scenario の内容。
 * @returns {string} OpenSpec 差分仕様の内容。
 */
function createDeltaSpec({
  kind = 'ADDED',
  requirement = 'アカウント作成',
  scenarioId = 'ACCOUNT-S002',
  manual = false,
} = {}) {
  return `## Purpose

利用者のアカウント操作を保証する。

## ${kind} Requirements

### Requirement: ${requirement}

システムは要求された結果を返さなければならない。

#### Scenario: 要求された結果を返す (${scenarioId})

${manual ? 'Tags: manual\n\n' : ''}- **WHEN** 利用者が操作する
- **THEN** 要求された結果が表示される
`;
}

test('全活動中差分の Scenario を主仕様とともに検査する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/specs/account/spec.md': createMainSpec(),
    'openspec/changes/add-account/specs/account/spec.md': createDeltaSpec(),
    'tests/account.test.ts':
      "test('[ACCOUNT-S001] display', () => {});\ntest('[ACCOUNT-S002] create', () => {});\n",
  });

  assert.equal(result.status, 0);
  assert.match(result.stdout, /coverage: OK/u);
});

test('Go の試験名にある Scenario 参照を検査対象に含める', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/specs/account/spec.md': createMainSpec(),
    'packages/backend/account_test.go':
      'package backend\n\nfunc TestAccount(t *testing.T) { t.Run("[ACCOUNT-S001] display", func(t *testing.T) {}) }\n',
  });

  assert.equal(result.status, 0);
  assert.match(result.stdout, /coverage: OK/u);
});

test('--change は選択した差分だけを主仕様へ重ねる', () => {
  const result = runGuardInFixture(
    guardScriptPath,
    {
      'openspec/specs/account/spec.md': createMainSpec(),
      'openspec/changes/change-a/specs/account/spec.md': createDeltaSpec({
        kind: 'MODIFIED',
        requirement: 'アカウント表示',
        scenarioId: 'ACCOUNT-S010',
      }),
      'openspec/changes/change-b/specs/account/spec.md': createDeltaSpec({
        kind: 'MODIFIED',
        requirement: 'アカウント表示',
        scenarioId: 'ACCOUNT-S020',
      }),
      'tests/account.test.ts': "test('[ACCOUNT-S010] selected overlay', () => {});\n",
    },
    ['--change', 'change-a']
  );

  assert.equal(result.status, 0);
  assert.doesNotMatch(result.stderr, /ACTIVE_SPEC_CONFLICT/u);
});

test('異なる活動中 Change の物質的に異なる操作を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/specs/account/spec.md': createMainSpec(),
    'openspec/changes/change-a/specs/account/spec.md': createDeltaSpec({
      kind: 'MODIFIED',
      requirement: 'アカウント表示',
      scenarioId: 'ACCOUNT-S010',
    }),
    'openspec/changes/change-b/specs/account/spec.md': createDeltaSpec({
      kind: 'MODIFIED',
      requirement: 'アカウント表示',
      scenarioId: 'ACCOUNT-S020',
    }),
    'tests/account.test.ts': "test('[ACCOUNT-S020] latest overlay', () => {});\n",
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /ACTIVE_SPEC_CONFLICT/u);
});

test('実効仕様内で重複する Scenario ID を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/add-account/specs/account/spec.md': `${createDeltaSpec()}
### Requirement: アカウント削除

#### Scenario: アカウントを削除する (ACCOUNT-S002)

- **WHEN** 利用者が削除する
- **THEN** アカウントが削除される
`,
    'tests/account.test.ts': "test('[ACCOUNT-S002] account operation', () => {});\n",
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Duplicate Scenario ID 'ACCOUNT-S002'/u);
});

test('Tags: manual の Scenario は試験参照を要求しない', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/add-account/specs/account/spec.md': createDeltaSpec({ manual: true }),
  });

  assert.equal(result.status, 0);
});

test('自動化対象 Scenario の参照欠落を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/specs/account/spec.md': createMainSpec(),
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Missing test reference 'ACCOUNT-S001'/u);
});

test('実効仕様にない試験参照を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'tests/account.test.ts': "test('[ACCOUNT-S999] orphan', () => {});\n",
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Orphan test reference 'ACCOUNT-S999'/u);
});

test('MODIFIED は主仕様の旧 Scenario を実効仕様から置き換える', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/specs/account/spec.md': createMainSpec(),
    'openspec/changes/change-account/specs/account/spec.md': createDeltaSpec({
      kind: 'MODIFIED',
      requirement: 'アカウント表示',
      scenarioId: 'ACCOUNT-S010',
    }),
    'tests/account.test.ts':
      "test('[ACCOUNT-S001] obsolete', () => {});\ntest('[ACCOUNT-S010] current', () => {});\n",
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Orphan test reference 'ACCOUNT-S001'/u);
});

test('差分仕様に Purpose を要求する', () => {
  const result = runGuardInFixture(guardScriptPath, {
    'openspec/changes/add-account/specs/account/spec.md': createDeltaSpec().replace(
      '## Purpose',
      '## Context'
    ),
    'tests/account.test.ts': "test('[ACCOUNT-S002] create', () => {});\n",
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /## Purpose が必要/u);
});

test('存在しない --change の指定を拒否する', () => {
  const result = runGuardInFixture(guardScriptPath, {}, ['--change', 'missing-change']);

  assert.equal(result.status, 1);
  assert.match(result.stderr, /存在しません/u);
});

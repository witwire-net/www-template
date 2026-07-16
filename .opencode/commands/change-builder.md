---
description: 仕様設計/ドメイン設計から OpenSpec の推奨分割を提示し、承諾後に proposer を並列起動して changes を作成する。
---

The user wants to design and scaffold OpenSpec changes using a dedicated workflow.

Input (design brief)

```text
$ARGUMENTS
```

Goal

- /change-builder で入力された仕様設計・ドメイン設計を基に、プロジェクト全体を考慮した推奨 Spec 分割（capability）と推奨 Changes 分割（change-id + 依存関係 + 並列グループ）を提案する
- ユーザーが各 change の再構成済み意図と分割を承諾したら、`openspec/proposer` サブエージェントを呼び出して各 change を作成する（並列可能なものは並列）
- 全 change の完成（validate PASS）を確認して報告する

Hard rules

- このコマンド(この実行)では実装しない（TypeSpec/コード/生成物の変更は行わず、OpenSpec の change proposal を作るところまで）
- ただし Change のスコープ自体は「承認後に TypeSpec -> 生成 -> 実装 -> テスト/ビルドまで到達する一連」を含むものとして扱う（= tasks.md には実装までのチェックリストを必ず書く）
- proposal/tasks で「この提案フェーズでは変更なし」「実装は後続 change/後続フェーズ」など Change のスコープを縮める表現を入れない（実行スコープと Change スコープを混同しない）
- `generated/**` を手編集しない
- 既存の OpenSpec ルールに従う（`openspec/AGENTS.md`）

Preflight context

- !`openspec list || true`
- !`openspec list --specs || true`
- !`ls -la openspec/changes 2>/dev/null || true`
- !`ls -la openspec/specs 2>/dev/null || true`

Process

1. Load rules

- Read:
  - `AGENTS.md`
  - `openspec/AGENTS.md`
  - `openspec/project.md`

2. Interpret the design brief

- Extract:
  - Actors / entities / invariants
  - Requirements (MUST/SHALL) and scenarios
  - API/contract surface (if any)
  - Data model / domain boundaries
  - Open questions / decisions
- Treat solution-shaped terms as evidence to classify, not automatically as requirements
- Inspect repository evidence and separate observed facts, inferences, assumptions, and unresolved decisions
- For each proposed Change, prepare an Intent Card containing:
  - Actor / situation / problem / desired outcome / priority
  - Required outcomes / non-negotiable constraints / candidate means
  - Repository evidence with `path:line`
  - Falsification check and conclusion
  - Invariants / boundaries / observable success

3. Propose Spec splitting (capabilities)

- Propose a set of capabilities (single responsibility per capability)
- Reuse/extend existing capabilities when appropriate
- For each capability:
  - Purpose (1-2 lines)
  - Key requirements (titles only)
  - Dependencies (if any)

4. Propose Changes splitting

- Propose change-ids (kebab-case, verb-led: add-/update-/remove-/refactor-)
- For each change:
  - Scope
  - Touched capabilities
  - Risks / breaking notes
  - Dependencies on other changes

5. Ask for intent and split approval

- Show every Change's complete Intent Card before asking for approval.
- Use the `question` tool with options:
  - この分割で進める
  - 意図または分割を修正したい（自由入力で指示）
  - いったん中止
- Approval confirms both the displayed per-Change intents and the split. Do not infer confirmation from silence or from approval of a materially different earlier summary.

6. If approved, scaffold changes via subagents

- Group changes into parallelizable batches based on dependencies.
- For each batch, call `task` in parallel:
  - `subagent_type: openspec/proposer`
  - Prompt includes a YAML `ChangePlan` and `IntentConfirmation` per change.
  - `IntentConfirmation` includes `status: confirmed`, the exact approved intent summary, all request-term classifications, and the owner's approval response.

7. Completion check and report

- For every change-id:
  - Ensure `openspec validate <id> --strict --no-interactive` is PASS
- Finally run:
  - `openspec validate --all --strict --no-interactive`
- Report:
  - Created/updated change-ids
  - Any remaining open questions (should be non-blocking)
  - Next steps (approval gate -> implementation)

Begin now.

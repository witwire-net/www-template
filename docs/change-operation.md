# 変更運用

本書は、変更の進め方、OpenSpec の利用境界、UX 方針、レビュー深度を定める一次資料です。コーディング規則の機械的な強制内容は `CODING_STANDARDS.md`、開発者向け手順は `CONTRIBUTING.md` を参照してください。

## 基本原則

OpenSpec は、現在および変更後に利用者や外部契約から観測できる振る舞いを永続的に管理する契約です。実装全体を先に分解した基本計画、ファイル一覧、補助処理一覧、試験階層別の作業表にはしません。

変更では、次の三つを独立に決めます。

1. `Operation Lane`: 変更をどの運用経路で進めるか。
2. `UX Mode`: 利用者に見える体験をどのように扱うか。
3. `Review Depth`: どの深さで独立レビューするか。

たとえば、内部構造を大きく変えながら画面体験を維持する変更は、`Operation Lane: ARCHITECTURE`、`UX Mode: CONTINUITY` です。運用区分から UX モードを推測してはいけません。

## Operation Lane

### `DIRECT`

観測可能な振る舞いと物質的なアーキテクチャ判断を変更しない作業です。文書の誤記修正、既存契約どおりに実装を直す局所的な不具合修正、生成や整形など、永続的な振る舞い契約を追加・変更しない作業が該当します。

- OpenSpec Change は不要です。
- プルリクエストの `OpenSpec Change` と `Scenario IDs` には、理由を添えた `なし` を記載できます。
- 作業中に振る舞いの変更または物質的な構造判断が必要だと判明した場合は、実装を続ける前に `BEHAVIOR` または `ARCHITECTURE` へ変更します。

### `BEHAVIOR`

利用者、呼び出し元、運用上の外部契約から観測できる結果を追加、変更、削除、改名する作業です。

- `behavior-change` スキーマの OpenSpec Change が必須です。
- `proposal.md`、差分仕様、`tasks.md` を使用します。
- Requirement と Scenario には観測可能な終端状態だけを記載します。
- 実装方式や内部構造は仕様へ記載しません。

### `ARCHITECTURE`

責務境界、依存方向、データ所有権、セキュリティ境界、移行、切り戻し、重大な障害形態など、物質的な内部構造を変更する作業です。

- `architecture-change` スキーマの OpenSpec Change が必須です。
- `proposal.md`、差分仕様、`design.md`、`tasks.md` を使用します。
- 差分仕様には、構造変更後も成立すべき観測可能な振る舞いを記載します。
- `design.md` には物質的な判断だけを記載し、詳細な実装手順へ分解しません。

## UX Mode

UX モードは運用区分とは別に判定します。ただし、`SHAPE` は利用者に見える体験を実質的に変えるため、観測可能な振る舞いを変更しない `DIRECT` とは組み合わせません。

### `NONE`

利用者に見える画面、操作、文言、情報階層を変更しません。OpenSpec Change を使う場合は、利用者に見える変更がない根拠を `## UI / UX Impact` に記載します。

### `CONTINUITY`

特定した既存体験を維持します。OpenSpec Change の `## UI / UX Impact` に `### Continuity Source` を置き、維持対象となる実装済み画面、既存の操作、承認済み資料などの根拠を記録します。

### `SHAPE`

利用者に見える体験の方向性を新たに定めます。OpenSpec Change の `## UI / UX Impact` に `### Primary User Task` と `### UX Direction` を置き、利用者が完了したい中心作業と目指す体験を、差分仕様を書く前に確定します。

UX の方向付けは必要な変更だけで行います。`SHAPE` でない変更へ、新しい画面構成や視覚表現を便宜的に持ち込みません。

## UI 変更の実証

利用者に見える UI を実際に変更する場合は、プロダクトデザイナーが既存製品、中心作業、承認済み UX 方針、共通 UI を確認し、実装可能な画面として設計します。実装後は必ず実際のブラウザで、デスクトップとモバイルの操作、表示、アクセシビリティを確認します。

画像生成による UI モックアップは、方向性の比較や会話を助ける任意の非契約証跡です。Requirement、Scenario、承認済み UX 方針、実装済み画面、実ブラウザでの確認結果を置き換えません。

プルリクエストでは、実際の UI / UX 変更がある場合に `Desktop Before`、`Desktop After`、`Mobile Before`、`Mobile After` の画像をすべて添付します。この要件は UX モードの選択とは別に、実際の変更内容から判定します。

## OpenSpec Change

Change のひな形と OpenCode のライフサイクルコマンドは、OpenSpec 自身が生成します。スキーマ選択と計画ルートの情報を正しく保つため、Change は必ず CLI から作成します。

```bash
scripts/devcontainer/run.sh pnpm openspec new change <change-id> --schema behavior-change
scripts/devcontainer/run.sh pnpm openspec new change <change-id> --schema architecture-change
```

OpenSpec `1.8.0` の `new change` は `openspec/config.yaml#schema` を Change 作成時の既定値として参照しないため、運用区分に対応する `--schema` を必ず指定します。`openspec/changes/<change-id>` を手作業で作成してはいけません。OpenSpec の更新後は `pnpm gen:openspec` で OpenCode の公式コアコマンドとスキルを同時に再生成し、`.opencode/commands/opsx-*.md` と `.opencode/skills/openspec-*/SKILL.md` を手編集しません。リポジトリ固有の補足スキルは `.opencode/skills/openspec/` に分離します。

### 提案

`proposal.md` は依頼の権威ある解釈です。依頼文をそのまま実装指示とみなさず、成果、成果の制約、必須手段、候補手段へ分類します。

- `Intent-Resolution: REQUEST_SUFFICIENT`: 依頼自体が重要な曖昧さをすべて解決している場合。
- `Intent-Resolution: OWNER_CONFIRMED`: 再構成した意図を所有者が明示確認した場合。
- `Intent-Resolution: DRAFT`: 振る舞い、外部契約、アーキテクチャ、セキュリティ、データ、依存関係、対象範囲を変え得る曖昧さが残る場合。

`DRAFT` の間は、差分仕様、設計、作業パッケージを作成しません。

### 永続的な振る舞い契約

主仕様は `openspec/specs/<capability>/spec.md` に置きます。活動中の Change は `openspec/changes/<change-id>/specs/<capability>/spec.md` に差分を持ちます。

Requirement と Scenario は、利用者または外部契約から観測できる終端状態だけを表します。特定のファイル、パッケージ、関数、補助処理、移行作業、試験階層を製品要件として記載しません。

### Scenario と試験の追跡

すべての Scenario 見出しは、`#### Scenario: ... (CAPABILITY-S001)` の形式で安定した識別子を持ちます。自動試験は題名へ `[CAPABILITY-S001]` を含めます。自動化できない場合は Scenario の近くに `Tags: manual` を記載します。

`scripts/openspec/verify-scenario-coverage.mjs` は、主仕様へすべての活動中差分を重ねた実効仕様を既定で検査します。これにより、同期やアーカイブの前でも識別子の重複、試験参照の欠落、孤立した試験参照、活動中 Change 間の競合を検出します。

一つの Change に限った実効仕様を確認する場合は、次を実行します。

```bash
scripts/devcontainer/run.sh pnpm lint:openspec:scenario -- --change <change-id>
```

`--change` は対象 Change と主仕様の組み合わせへ集中するための選択機能です。全活動中 Change 間の競合確認を置き換えないため、最終確認では引数なしの検査も実行します。

## 段階的な実行計画

`tasks.md` は、顧客成果または物質的な設計判断を実装可能なまとまりにした作業パッケージ台帳です。各項目は `- [ ] WP<number>: <成果>` とし、`Covers` と `Completion Evidence` を持ちます。

OpenSpec では実装全体を事前に詳細分解しません。実装時に、現在の作業パッケージ、リポジトリの実態、直前の検証結果を読み、次の安全な変更と確認を決めます。ファイル単位、補助処理単位、試験階層単位の詳細は、実行時に必要な範囲だけ計画し、永続的な基本計画として残しません。

作業パッケージの完了条件は、リポジトリ内または CI で再現できる証跡に限定します。リリース、デプロイ、外部環境の操作、認証情報へのアクセス、外部承認を含めません。

## Review Depth

### `STANDARD`

通常の変更に対する既定の独立レビューです。次を確認します。

- 依頼、運用区分、UX モード、変更内容が一致していること。
- 適用される Requirement と Scenario を満たすこと。
- セキュリティ、依存方向、型、生成物、試験の規則を破っていないこと。
- UI 変更では、プロダクトデザイナーの関与と実ブラウザ確認の証跡があること。
- 不要なコード、未使用機能、仮置きが残っていないこと。

### `DEEP`

重大な誤りが顧客、安全性、データ、外部契約、運用継続性へ広く影響する変更に使用します。`STANDARD` の確認に加え、関連する呼び出し経路、活動中 Change との相互作用、信頼境界、データ整合性、移行と切り戻し、障害形態、否定経路、残存リスクを追跡します。

次のいずれかに該当する場合は `DEEP` を選びます。

- 認証、認可、秘密情報、個人情報、金銭、重要な入力境界を変更する。
- データ所有権、永続化形式、移行、切り戻しを変更する。
- 公開 API または外部契約へ破壊的な影響がある。
- 複数の層や機能領域にまたがる物質的なアーキテクチャ変更である。
- 複数の活動中 Change と同じ Requirement または責務境界へ影響する。
- 所有者が明示的に深いレビューを要求する。

レビュー深度は運用区分から自動決定しません。`DIRECT` でもセキュリティ上重大なら `DEEP`、`ARCHITECTURE` でも限定的で十分な証拠が揃うなら `STANDARD` を選べます。

## プルリクエスト記録

すべてのプルリクエストに次を記録します。

- `Operation Lane`: `DIRECT`、`BEHAVIOR`、`ARCHITECTURE` のいずれか。
- `UX Mode`: `NONE`、`CONTINUITY`、`SHAPE` のいずれか。
- `Review Depth`: `STANDARD`、`DEEP` のいずれか。
- `OpenSpec Change`: `BEHAVIOR` と `ARCHITECTURE` では必須。`DIRECT` では理由付きの `なし` を使用可能。
- `Scenario IDs`: `BEHAVIOR` と `ARCHITECTURE` では一件以上必須。`DIRECT` では理由付きの `なし` を使用可能。

## 検証入口

```bash
scripts/devcontainer/run.sh pnpm gen:openspec
scripts/devcontainer/run.sh pnpm lint:openspec
```

`pnpm lint:openspec` は二スキーマ、全成果物、提案、Scenario と試験の追跡、作業パッケージと設計の対象範囲を一括検証します。全体確認では `scripts/devcontainer/run.sh pnpm lint` を実行します。

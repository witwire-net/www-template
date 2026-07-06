---
name: generate-image
description: Use scripts/generate-image.mjs to generate raster images through Codex CLI and Codex's built-in image generation tool; supports UI mockups, product mockups, hero visuals, edits, references, and optional wireframe input.
---

# Generate Image

この Skill は、**`.opencode/skills/generate-image/scripts/generate-image.mjs` の CLI フォーム欄を正しく埋めるための運用手順**です。

画像生成 prompt を手で組み立てたり、`codex exec` を直接書いたりしないでください。通常経路では必ず `generate-image.mjs` を呼び出します。

## Primary Rule

画像生成・画像編集・画像 mockup が必要なときは、次を入口にします。巨大な自由文 prompt を 1 つ渡すのではなく、成果物仕様書の各欄を CLI 引数として埋めます。

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template <template> \
  --prompt "<成果物タイプ + 視覚モード + 用途>" \
  --purpose "<用途と伝えるべきこと>" \
  --canvas "<比率、媒体、余白、safe area>" \
  --subject "<主対象>" \
  --composition "<配置、優先順位、視点>" \
  --style "<視覚言語>" \
  --details "<素材、状態、既存 design との整合>" \
  --constraint "<禁止事項や崩してはいけない条件>"
```

`generate-image.mjs` が次を一元管理します。

- テンプレート選択。
- CLI フォーム欄から GPT-Image-2 向け成果物仕様書型 prompt を組み立てること。
- 未入力欄への template 既定値と安全制約の補完。
- `ui-mockup --wireframe` の wireframe 要約。
- Codex CLI への guardrail 付与。
- `codex exec` の安全な起動引数。
- 出力 path の決定と存在確認。

## When To Use

- Codex CLI 経由で GPT-Image-2 相当の raster image を生成したいとき。
- UI mockup、product mockup、landing hero、広告、図解、商品写真、透明切り抜き用素材、画像編集を作りたいとき。
- `ui-mockup` で任意の wireframe JSON/HTML を構造参照にしたいとき。
- 長文ポエムではなく、成果物仕様書型 prompt で低リトライにしたいとき。

## When Not To Use

- SVG、HTML、CSS、canvas、Svelte コンポーネントなど、コードネイティブな成果物が正しいとき。
- 既存アイコン、既存ロゴ、既存 UI 実装を直接編集する方が正確なとき。
- `quality`、`size`、`output_format`、`request_id`、コストログを API パラメータとして厳密に制御したいとき。
- 証明書、領収書、契約書、金融画面、本人確認書類など、偽文書に見える画像を作る用途。

## Prerequisites

- `codex` CLI が PATH にあること。
- `codex login` が完了していること。
- Node.js で `generate-image.mjs` を実行できること。

確認 command:

```bash
codex --version
codex login status
```

`codex` が PATH に無い場合は、`CODEX_BIN` または `GENERATE_IMAGE_CODEX_BIN` に Codex 実行ファイル path を指定します。

## Workflow

1. ユーザーの目的から template を選びます。
2. 必要な根拠を調べ、`--prompt`、`--purpose`、`--canvas`、`--subject`、`--composition`、`--style`、`--details`、`--constraint` に分解します。
3. 文字が必要なら `--text` と `--typography` を使い、exact text、出現回数、余計な文字の禁止を明示します。
4. 編集なら `--change-only`、`--preserve`、`--physical-realism` を使います。
5. 迷う場合や wireframe 入力がある場合は、先に `--dry-run` で prompt を確認します。
6. ユーザーが画像生成を求めている場合だけ、dry-run なしで実生成します。
7. 生成後は出力 path を報告します。
8. 生成画像は user data として扱い、明示なしに削除・上書き・commit しません。

## Command Examples

基本形:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template ui-mockup \
  --prompt "Create a realistic web app dashboard mockup for a small-team groupware product." \
  --purpose "Product screenshot-style visual for reviewing UI direction and product realism." \
  --canvas "1536x1024 landscape. Practical desktop UI proportions with readable spacing." \
  --subject "A polished dashboard showing calendar, tasks, notes, and messages as practical workspace surfaces." \
  --composition "Left navigation, central schedule and task work area, secondary updates panel, clear primary action hierarchy." \
  --style "Existing product UI language, restrained SaaS typography, subtle borders, coherent token-based color system." \
  --details "Use implementable spacing, shared component-like cards, clear focus order, and no decorative filler." \
  --constraint "No authentication screen." \
  --constraint "No random analytics charts unless requested." \
  --out "$HOME/Pictures/codex-images/groupware-dashboard.png" \
  --quality medium \
  --size 1536x1024
```

dry-run:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template ui-mockup \
  --prompt "Create a realistic web app dashboard mockup for a premium B2B SaaS product." \
  --purpose "Dry-run to inspect the structured image-generation prompt before spending image quota." \
  --subject "A dense but readable dashboard with navigation, primary content, and supporting status panels." \
  --dry-run
```

wireframe を任意入力にする `ui-mockup`:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template ui-mockup \
  --prompt "Create a realistic web app UI mockup for the specified dashboard page." \
  --purpose "Product screenshot-style mockup for OpenSpec design review." \
  --canvas "1536x1024 landscape. Keep safe margins and preserve the wireframe's main page regions." \
  --subject "The page described by the wireframe, rendered as a shippable product UI rather than a low-fidelity sketch." \
  --composition "Follow the wireframe structure for navigation, content groups, actions, and screen-level hierarchy." \
  --style "Match the repository's existing frontend design language, shared UI components, spacing rhythm, typography, and token-based colors." \
  --details "Preserve information architecture while replacing gray scaffold boxes with production-quality component treatment." \
  --constraint "Do not copy low-fidelity wireframe styling." \
  --constraint "No fake logos or decorative badges." \
  --wireframe path/to/dashboard.wireframe.json \
  --out "$HOME/Pictures/codex-images/dashboard-ui.png" \
  --quality medium \
  --size 1536x1024 \
  --iteration-target "layout hierarchy and product realism"
```

参照画像を渡す例:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template reference-composite \
  --prompt "premium product hero using the product shape from Image 1 and the lighting mood from Image 2" \
  --image ./product.png \
  --image-role "Image 1: base product photo; preserve shape, label, color, and camera angle" \
  --image ./lighting-reference.png \
  --image-role "Image 2: lighting and background reference only" \
  --out "$HOME/Pictures/codex-images/product-hero.png"
```

## CLI Form Fields

- `--prompt`: 第 1 行。成果物タイプ、視覚モード、用途を固定します。
- `--purpose`: どこで使い、何を伝える画像か。
- `--canvas`: 比率、サイズ、媒体、crop、safe area、余白。
- `--subject`: 主対象。UI、人物、商品、図、背景など。
- `--composition`: 配置、視点、優先順位、情報階層。
- `--style`: 写真、UI、広告、ポスター、漫画、ベクター、3D などの視覚言語。
- `--text`: 画像内文字の exact text、出現回数、余計な文字の禁止。
- `--typography`: 書体、太さ、配置、可読性。
- `--details`: 素材、色、照明、UI state、既存 design との整合。
- `--preserve`: 編集時または参照時に維持するもの。
- `--change-only`: 編集時に変更する範囲。
- `--physical-realism`: 編集時の影、反射、edge、scale など。
- `--constraint`: 1 件ずつ追加する禁止事項。複数回指定できます。
- `--constraints`: 改行または semicolon 区切りでまとめて制約を渡します。

## Template Selection

- `general`: 汎用画像生成。
- `ui-mockup`: 高忠実度な UI 画面画像。任意で `--wireframe` を指定できます。
- `product-mockup`: 商品・サービスを魅力的に見せる product / marketing mockup。
- `landing-hero`: LP の hero visual。
- `ad-creative`: 広告・SNS クリエイティブ。
- `infographic`: 図解・インフォグラフィック。
- `product-shot`: 商品写真風の画像。
- `transparent-cutout`: 後処理で背景除去しやすい単色背景の素材。
- `image-edit`: 入力画像を編集する用途。少なくとも 1 つの `--image` が必要です。
- `reference-composite`: 複数画像の役割を明示して合成・参照する用途。

## ui-mockup With --wireframe

`--wireframe` は `ui-mockup` の任意入力です。

`generate-image.mjs` は wireframe JSON/HTML を prompt に直貼りしません。画面名、viewport、layout tree、element type、table columns、actions を抽出し、画像生成向けの短い UI brief に変換します。

wireframe は構造参照です。低忠実度の見た目、灰色 box、汎用 outline、均一な矩形、scaffold label を完成 UI の style として再現させません。この制御も `mjs` 側の prompt builder が担当します。

## Quality And Size

- `--quality low`: 方向性探索、thumbnail、ラフ案。
- `--quality medium`: 標準。LP 素材、UI 案、広告案、通常の product mockup。
- `--quality high`: 最終、文字多め、商品ラベル、図解、細部重視。
- `--size`: `auto` または `1536x1024` などの `WIDTHxHEIGHT`。

Codex built-in tool 経由では、`quality` と `size` は API パラメータ保証ではありません。`mjs` が `Quality target:` と `Canvas:` として prompt に入れます。

## Safety Rules

- 通常経路では `generate-image.mjs` を使い、手書きの `codex exec` を使いません。
- Codex には built-in image generation tool だけを使わせます。
- OpenAI API、curl、Python SDK、独自 API runner は使わせません。
- SVG、HTML、CSS、canvas、placeholder image で代替させません。
- API key を chat や config に書かせません。
- 透明背景は `gpt-image-2` で直接指定できない前提にします。必要な場合は `transparent-cutout` で単色背景素材を作り、後処理で背景除去します。

## Bundled Resources

- `scripts/generate-image.mjs`: この Skill の実行入口。
- `references/prompting.md`: GPT-Image-2 向け prompt 方針。通常は読むだけで、実行は `mjs` に任せます。

## Troubleshooting

- `codex` が見つからない場合は `codex --version` を確認し、必要なら `CODEX_BIN` または `GENERATE_IMAGE_CODEX_BIN` を指定します。
- `codex login status` が失敗する場合は、対話的に `codex login` を実行してから再試行します。
- 出力ファイルが作られない場合は、同じ引数に `--dry-run` を付けて prompt と command を確認します。
- 文字が崩れる場合は、`--quality high` を使い、ユーザー prompt に exact text、出現回数、no extra text、no duplicate text を含めます。

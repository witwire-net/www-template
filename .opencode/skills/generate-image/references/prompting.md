# Generate Image Prompting Reference

この reference は、`generate-image` Skill が Codex CLI の built-in image generation tool に渡す prompt を安定させるための実務ルールです。

## Core Principle

最適な prompt は、長文ポエムではなく成果物仕様書です。

目標は、1 回で芸術的に良い画像を出すことではありません。用途に合う画像を、低リトライ、低コスト、編集耐性ありで生成することです。

## Prompt Shape

```text
Create a [visual mode] [deliverable type] for [specific use case].

Purpose:
[Where this image will be used and what it must communicate.]

Canvas:
[Aspect ratio, size, orientation, crop, safe area, whitespace.]

Subject:
[Main object/person/UI/scene. Be concrete.]

Composition:
[Placement, camera angle, hierarchy, spacing, focal point.]

Style:
[Photo/design/illustration/UI/poster style. Mention medium, texture, realism level.]

Text:
[Exact text if needed. Use quotes. State exactly once. State no extra text.]

Details:
[Materials, colors, lighting, props, UI states, environment.]

Preserve:
[For edits: identity, geometry, label, lighting, framing.]

Constraints:
[No clutter, no fake logos, no watermark, no duplicate text, no undesired UI patterns.]

Iteration target:
[What this generation should optimize: layout, text, realism, brand fit, etc.]
```

## CLI Form Mapping

`generate-image.mjs` は、上の成果物仕様書を CLI フォーム欄として受け取ります。AI に巨大な自由文を渡すのではなく、各欄を分けて渡すことで、毎回同じ順序と粒度の prompt を組み立てます。

| Prompt section   | CLI option                       |
| ---------------- | -------------------------------- |
| First line       | `--prompt`                       |
| Purpose          | `--purpose`                      |
| Canvas           | `--canvas` と `--size`           |
| Subject          | `--subject`                      |
| Composition      | `--composition`                  |
| Style            | `--style`                        |
| Text             | `--text`                         |
| Typography       | `--typography`                   |
| Details          | `--details`                      |
| Preserve         | `--preserve`                     |
| Constraints      | `--constraint` / `--constraints` |
| Iteration target | `--iteration-target`             |

`--prompt` は短い雰囲気指定ではなく、成果物タイプ、視覚モード、用途を固定する第 1 行として書きます。

## Quality Policy

- `low`: 方向性探索、thumbnail、ラフ案。
- `medium`: 標準。LP 素材、UI 案、広告案、通常の product mockup。
- `high`: 最終、文字多め、商品ラベル、図解、細部重視。
- `auto`: Codex 側に任せる場合。

Codex built-in tool 経由では、`quality` は API パラメータ保証ではありません。`Quality target:` として prompt に入れます。

## Size Policy

推奨サイズ:

- `1024x1024`: square draft。
- `1536x1024`: 標準 landscape。
- `1024x1536`: 標準 portrait。
- `2048x1152`: 2K landscape。
- `2048x2048`: 2K square。
- `3840x2160`: 4K landscape、必要時だけ。
- `2160x3840`: 4K portrait、必要時だけ。
- `auto`: Codex 側に任せる場合。

Codex built-in tool 経由では、`size` も API パラメータ保証ではありません。`Canvas:` として prompt に入れます。

## Text In Images

文字が必要な場合は、内容ではなく typography specification として書きます。

```text
Text:
Include ONLY this headline, exactly once, verbatim:
"Work, without the noise"

Typography:
Bold modern sans-serif.
High contrast.
Centered in the upper third.
Readable at thumbnail size.

Constraints:
No other text.
No duplicate text.
No misspellings.
No fake logos.
No watermark.
```

日本語の場合:

```text
Text:
Include ONLY this Japanese headline, exactly once:
"情報を減らすほど、仕事は進む"

Typography:
Large, clean Japanese gothic type.
High contrast.
Horizontal writing.
No furigana.
No extra English words.
No duplicate text.
```

## Editing

編集では、何を変えるかより、何を変えないかを明確にします。

```text
Edit the input image.

Change only:
[One clear change.]

Preserve exactly:
[Identity, face, product geometry, label, camera angle, lighting, shadows, layout.]

Physical realism:
[Scale, contact shadows, reflections, texture, edge blending.]

Constraints:
[No redesign, no retouching, no extra objects, no logo drift, no watermark.]
```

## Multiple Images

複数画像は、単なる reference ではなく役割で渡します。

```text
Image 1:
Base image. Preserve shape, label, angle, and proportions.

Image 2:
Lighting and background reference only.

Task:
Create the final image using the product from Image 1 and the lighting mood from Image 2.

Preserve:
Product label text, product geometry, camera angle, packaging color.

Do not:
Do not copy logos or text from Image 2.
Do not redesign the product.
Do not invent new label text.
```

## ui-mockup With Wireframe

`ui-mockup` は UI 画面そのものを高忠実度な raster image として生成する用途です。

`--wireframe` が指定された場合、wireframe は layout and information architecture guidance としてだけ扱います。低忠実度な visual style はコピーしません。

```text
Create a realistic web app UI mockup for [specific product/use case].

Purpose:
Product screenshot-style visual for reviewing UI direction.

Canvas:
1536x1024 landscape.
Use practical desktop UI proportions and readable spacing.

Wireframe reference:
Use the wireframe as layout and information architecture guidance only.
Preserve the major sections, hierarchy, content groups, navigation roles, and primary actions.
Do not copy the wireframe's low-fidelity visual style.
Do not render gray placeholder boxes, generic outlines, equal-weight rectangles, or scaffold labels as final UI styling.

Screen structure:
[summary generated from wireframe]

Style:
Realistic shippable product UI.
Readable typography.
Practical component hierarchy.
Coherent color system.
Implementable spacing.
Not concept art.

Constraints:
No fake logos.
No watermark.
No decorative badges.
No unreadable dense microtext.
No random analytics charts unless requested.

Iteration target:
Layout hierarchy and product realism.
```

## Template Notes

### ui-mockup

- 冒頭は成果物タイプで固定します。
- Concept art 語彙を避けます。
- 実装可能な spacing、hierarchy、typography、component structure を重視します。
- Wireframe がある場合は、構造参照として扱います。

### product-mockup

- 主役は商品・サービスの見せ方です。
- デバイス、パッケージ、広告構図、光、素材感、余白を明示します。
- UI が含まれる場合でも、UI は画面内 content であり、主目的は marketing visual です。

### landing-hero

- Above-the-fold で使う画像として評価させます。
- Negative space、copy area、CTA 周辺の余白を明示します。
- 不要な badges、fake logos、random charts を禁止します。

### transparent-cutout

- `gpt-image-2` は transparent background を直接サポートしません。
- 単色 background で生成し、後処理で background removal する前提にします。
- 影、gradient、floor plane、reflection、texture を背景に入れさせません。

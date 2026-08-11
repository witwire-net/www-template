# Generate Image Prompting Reference

Use this reference to keep instructions sent through `generate-image.mjs`
consistent, efficient, and editable.

## Core Principle

Write a deliverable specification, not a long atmospheric prompt. Optimize for
fit to purpose, low retry count, and safe iteration.

## Prompt Shape

```text
Create a [visual mode] [deliverable type] for [specific use case].

Purpose:
[Where the image is used and what it communicates.]

Canvas:
[Aspect ratio, size, orientation, crop, safe area, and whitespace.]

Subject:
[Concrete primary object, person, UI, diagram, or scene.]

Composition:
[Placement, camera angle, hierarchy, spacing, and focal point.]

Style:
[Medium, texture, realism, and design language.]

Text:
[Exact quoted text, exactly once, with no extra text.]

Details:
[Materials, colors, lighting, props, states, and environment.]

Preserve:
[For edits: identity, geometry, labels, lighting, and framing.]

Constraints:
[One explicit prohibition per line.]

Iteration target:
[Layout, text, realism, brand fit, or another explicit optimization.]
```

## CLI Mapping

| Prompt section   | CLI option                      |
| ---------------- | ------------------------------- |
| First line       | `--prompt`                      |
| Purpose          | `--purpose`                     |
| Canvas           | `--canvas`, `--size`            |
| Subject          | `--subject`                     |
| Composition      | `--composition`                 |
| Style            | `--style`                       |
| Text             | `--text`                        |
| Typography       | `--typography`                  |
| Details          | `--details`                     |
| Preserve         | `--preserve`                    |
| Constraints      | `--constraint`, `--constraints` |
| Iteration target | `--iteration-target`            |

`--prompt` is the precise first-line deliverable instruction, not a mood-only
phrase.

## Artifact Recovery

Codex generates the image; the Node.js script places it at the requested output
path.

- Run `codex exec` with `--json`.
- Extract generated paths under `generated_images` from JSONL events.
- Only when JSONL is insufficient, search `CODEX_HOME/generated_images` and
  `~/.codex/generated_images` for artifacts created after execution began.
- Copy only when exactly one artifact is identified.
- Fail explicitly when multiple candidates exist.
- Never weaken sandboxing or use bypass flags.

## Quality and Size

- `low`: direction exploration
- `medium`: normal UI, ad, and product mockups
- `high`: final text and detailed output
- `auto`: Codex selects the target

Common sizes are `1024x1024`, `1536x1024`, `1024x1536`, `2048x1152`, and
`2048x2048`. Treat quality and size as prompt guidance when the built-in tool
does not expose them as strict API arguments.

## Text

Specify visible text as typography requirements:

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

For Japanese text, provide the exact Japanese string and specify Japanese font
style, writing direction, and no extra English words.

## Editing

State both the change and preservation boundary:

```text
Edit the input image.

Change only:
[One clear change.]

Preserve exactly:
[Identity, geometry, label, camera angle, lighting, shadows, and layout.]

Physical realism:
[Scale, contact shadows, reflections, texture, and edge blending.]

Constraints:
[No redesign, retouching, extra objects, logo drift, or watermark.]
```

## Multiple Images

Assign an explicit role to every input:

```text
Image 1:
Base image. Preserve shape, label, angle, and proportions.

Image 2:
Lighting and background reference only.

Task:
Use the product from Image 1 with the lighting mood from Image 2.

Do not:
Copy logos or text from Image 2, redesign the product, or invent label text.
```

## Template Guidance

### UI Mockup

- Generate a realistic product screen for comparing UI direction.
- Prioritize implementable spacing, hierarchy, typography, component structure,
  reachable states, and current product language.
- Avoid concept-art vocabulary, fake logos, decorative badges, unreadable
  microtext, and random analytics.
- Treat the image as optional non-contract evidence. Never replace implemented
  UI or browser review with it.

### Product Mockup

- Keep the product or service as the primary subject.
- Specify device or package, marketing composition, lighting, materials, and
  whitespace.
- UI inside a device is supporting content, not the primary deliverable.

### Landing Hero

- Evaluate it above the fold.
- Reserve negative space for copy and the primary action.
- Prohibit irrelevant badges, fake logos, and random charts.

### Transparent Cutout

- Assume direct transparency is unavailable.
- Generate on a solid removable background.
- Keep shadows, gradients, floor planes, reflections, and texture out of the
  background.

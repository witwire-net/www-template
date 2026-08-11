---
name: generate-image
description: Use scripts/generate-image.mjs to generate raster images through Codex CLI and its built-in image tool. Supports UI mockups, product mockups, hero visuals, edits, and image references.
---

# Generate Image

Use `.opencode/skills/generate-image/scripts/generate-image.mjs` as the normal
entrypoint for Codex image generation and editing. Do not assemble one long
free-form prompt or call `codex exec` directly.

## Primary Rule

Express the deliverable as separate CLI fields:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template <template> \
  --prompt "<deliverable type, visual mode, and use>" \
  --purpose "<where it is used and what it communicates>" \
  --canvas "<ratio, medium, whitespace, and safe area>" \
  --subject "<primary subject>" \
  --composition "<placement, priority, and viewpoint>" \
  --style "<visual language>" \
  --details "<materials, states, and product-system fit>" \
  --constraint "<one prohibition or preservation rule>"
```

The script owns template defaults, safety constraints, Codex invocation,
generated-artifact recovery, output placement, and existence checks.

## Use This Skill For

- Raster image generation or editing through Codex CLI
- UI and product mockups, landing heroes, ads, infographics, product shots, and
  cutout-ready assets
- Multi-image composition with explicit roles for each reference

UI mockups are optional, non-contract evidence for comparing direction. They do
not replace OpenSpec Requirements or Scenarios, approved UX direction, the
implemented surface, or real browser review.

## Do Not Use It For

- SVG, HTML, CSS, Canvas, React components, or other code-native output
- Work better handled by directly editing an existing icon, logo, or UI
- Fake certificates, receipts, contracts, financial screens, or identity
  documents

## Prerequisites

- `codex` is available on `PATH`
- `codex login` is complete
- Node.js can run `generate-image.mjs`

```bash
codex --version
codex login status
```

Use `CODEX_BIN` or `GENERATE_IMAGE_CODEX_BIN` when the executable is not on
`PATH`.

## Workflow

1. Select the template that matches the deliverable.
2. Ground the request and split it across the structured fields.
3. Use `--text` and `--typography` for exact visible text and prohibit extra or
   duplicate text.
4. For edits, use `--change-only`, `--preserve`, and `--physical-realism`.
5. Use `--dry-run` when the instruction needs inspection.
6. Generate only when the user requested image generation.
7. Report the output path.
8. Treat generated images as user data. Do not delete, overwrite, or commit them
   without explicit instruction.

## Examples

UI mockup:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template ui-mockup \
  --prompt "Create a realistic dashboard mockup for a small-team workspace." \
  --purpose "Compare product UI direction and implementation realism." \
  --canvas "1536x1024 landscape with practical spacing and safe margins." \
  --subject "Calendar, tasks, notes, and messages as working product surfaces." \
  --composition "Left navigation, central work area, secondary updates panel, clear primary action." \
  --style "Current product language, restrained typography, subtle borders, token-based color." \
  --details "Implementable spacing, coherent states, readable hierarchy, no decorative filler." \
  --constraint "No random analytics charts." \
  --out "$HOME/Pictures/codex-images/workspace-dashboard.png" \
  --quality medium \
  --size 1536x1024
```

Dry run:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template ui-mockup \
  --prompt "Create a realistic premium B2B dashboard mockup." \
  --purpose "Inspect the structured generation instruction before using image quota." \
  --subject "A dense but readable dashboard with navigation and supporting status." \
  --dry-run
```

Multiple references:

```bash
node .opencode/skills/generate-image/scripts/generate-image.mjs \
  --template reference-composite \
  --prompt "Create a premium hero using the product from Image 1 and lighting from Image 2." \
  --image ./product.png \
  --image-role "Image 1: preserve product shape, label, color, and camera angle" \
  --image ./lighting-reference.png \
  --image-role "Image 2: lighting and background reference only" \
  --out "$HOME/Pictures/codex-images/product-hero.png"
```

## Form Fields

- `--prompt`: deliverable type, visual mode, and use
- `--purpose`: placement and communication goal
- `--canvas`: ratio, dimensions, medium, crop, safe area, whitespace
- `--subject`: primary UI, person, product, diagram, scene, or background
- `--composition`: placement, viewpoint, priority, and hierarchy
- `--style`: photo, UI, ad, poster, illustration, vector, or 3D language
- `--text`: exact visible text, occurrence count, and no-extra-text rule
- `--typography`: face, weight, placement, and readability
- `--details`: materials, color, lighting, UI states, and product-system fit
- `--preserve`: elements unchanged in an edit or reference workflow
- `--change-only`: the exact edit boundary
- `--physical-realism`: scale, shadows, reflections, edges, and texture
- `--constraint`: one repeatable prohibition
- `--constraints`: newline- or semicolon-separated prohibitions

## Templates

- `general`: general raster generation
- `ui-mockup`: high-fidelity product UI image
- `product-mockup`: product or service marketing mockup
- `landing-hero`: above-the-fold landing visual
- `ad-creative`: advertising or social creative
- `infographic`: explanatory diagram
- `product-shot`: product-photography style image
- `transparent-cutout`: solid-background asset for later removal
- `image-edit`: edit at least one `--image`
- `reference-composite`: combine references with explicit roles

## Quality and Size

- `low`: direction exploration and thumbnails
- `medium`: normal UI, ad, and product mockups
- `high`: final images with text or fine detail
- `auto`: leave quality selection to Codex

Supported size guidance includes `1024x1024`, `1536x1024`, `1024x1536`,
`2048x1152`, and `2048x2048`. The built-in image tool may not expose quality
and size as API parameters, so the script adds them to `Quality target:` and
`Canvas:` in the instruction.

## Safety

- Use the script, not handwritten `codex exec`.
- Instruct Codex to use only its built-in image generation tool.
- Let Node.js copy the generated artifact to the requested output path.
- Never call the OpenAI API, curl, Python SDK, or a custom API runner.
- Never substitute SVG, HTML, CSS, Canvas, or placeholder images.
- Never write API keys to chat or configuration.
- For transparent output, generate a solid-background cutout and remove the
  background in a separate step.

## Resources

- `scripts/generate-image.mjs`: execution entrypoint
- `references/prompting.md`: structured prompting policy

## Troubleshooting

- Check `codex --version` and configure `CODEX_BIN` when Codex is unavailable.
- Run `codex login` interactively when login status fails.
- Add `--dry-run` to inspect the prompt and command when no output is created.
- The script fails rather than copying when artifact recovery finds multiple
  candidates.
- For text quality, use `--quality high`, exact text, one occurrence, and
  explicit no-extra/no-duplicate constraints.

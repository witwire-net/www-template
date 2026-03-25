---
name: gpt-ux
description: Use this skill for tasks involving visually strong landing pages, websites, apps, dashboards, prototypes, demos, or game UIs. Optimized for GPT-5.4 and smaller or lower-reasoning coding models. This skill enforces strong constraints, narrative structure, clear visual hierarchy, real content, restrained motion, a cardless-by-default composition style, and a distinctive aesthetic direction that avoids generic AI-generated design patterns.
---

# GPT-UX Skill

This skill summarizes the core fundamentals of UI/UX design and should be used during frontend design and implementation.

This skill is optimized for low-reasoning or medium-reasoning models.  
The model must be concrete, decisive, and output-oriented.  
Do not overthink. Do not drift into endless brainstorming.  
Make fast decisions from the rules below and build.  
These are not guidelines. They are rules and mandatory requirements.

## Primary Goal

Produce frontends that feel intentional, premium, current, and unmistakably designed.

Core principles:

- One strong idea per screen
- Clear visual hierarchy
- Minimal but meaningful copy
- Strong spacing rhythm
- Restrained use of color unless the concept requires otherwise
- Imagery that carries narrative weight
- No more than 2 to 3 memorable motions
- A clear and distinctive aesthetic point of view

Avoid:

- Generic SaaS card grids
- Weak brand presence
- Dashboards that are nothing more than stacked boxes
- Decorative gradients with no purpose
- Placeholder copy or filler text
- Over-explaining the UI inside the UI
- Generic AI-generated aesthetics
- Safe, default, interchangeable design choices

## Design Thinking

This skill guides the creation of distinctive, production-grade frontend interfaces that avoid generic "AI slop" aesthetics. Implement real working code with exceptional attention to aesthetic detail and creative choice.

The user provides frontend requirements: a component, page, application, or interface to build. They may also provide context about purpose, audience, brand, or technical constraints.

Before coding, understand the context and commit to a bold aesthetic direction:

- **Purpose**: What problem does this interface solve? Who uses it?
- **Tone**: Choose a clear and specific aesthetic direction. Examples include brutally minimal, maximalist chaos, retro-futuristic, organic, luxury, playful, editorial, brutalist, art deco, soft pastel, or industrial. Use these as inspiration, but commit to one direction that fits the context.
- **Constraints**: Respect technical requirements such as framework, performance, accessibility, design system, and maintainability.
- **Differentiation**: Identify what makes the interface unforgettable. Decide what one thing a user will remember after seeing it.

**Critical**: Choose a clear conceptual direction and execute it with precision. Bold maximalism and refined minimalism can both succeed. What matters is intentionality, not intensity.

Then implement working code, whether HTML, CSS, JavaScript, React, Vue, or another required stack, that is:

- production-grade and functional
- visually striking and memorable
- cohesive, with a clear aesthetic point of view
- meticulously refined in every detail

## Operating Mode

Always assume the following:

- Reasoning should be low by default
- Medium reasoning is allowed only when the UI is clearly complex
- Speed, clarity, and taste take priority over excessive exploration

Do not start writing code immediately.  
First define a concise design brief.  
Then build.

If the user has not provided enough direction, do not keep asking questions.  
Instead, infer a strong default and proceed.  
Only ask questions when missing information would block the work.

## Inputs That Must Be Defined Before Coding

Before producing code, write the following items in short bullet form.

1. visual thesis

- One sentence describing the mood, material quality, tone, and energy

2. content plan

- For landing pages: hero, support, detail, social proof if needed, final CTA
- For apps: workspace, navigation, secondary context, action areas

3. interaction thesis

- No more than 2 to 3 motion ideas
- Each motion must improve hierarchy, atmosphere, feedback, or focus

4. design system

- background
- surface
- primary text
- muted text
- accent
- border strategy
- display typography
- heading typography
- body typography
- spacing rhythm
- corner radius rule
- shadow rule

5. content source

- Use the user’s real copy when possible
- Otherwise, write realistic product copy that feels real
- Do not use placeholder thinking such as lorem ipsum

## Mandatory Rules

### Aesthetic Direction

- Every interface must have a clear aesthetic point of view.
- Avoid generic "AI slop" aesthetics at all times.
- Do not produce cookie-cutter layouts, predictable component arrangements, or context-free visual styling.
- Make creative choices that feel specific to the product, audience, and use case.
- No two designs should feel the same by default.
- Vary visual direction across outputs when the context changes.
- Intentionality matters more than trend-following.
- Match implementation complexity to the aesthetic vision.
- Maximalist designs may require richer structure, layered effects, and more elaborate motion.
- Minimalist or refined designs require restraint, precision, spacing discipline, and subtle detail.

### Composition

- Start from composition, not from components.
- The first viewport must read as a single composition, not as a pile of parts.
- Treat the first viewport like a poster, not like a document.
- Each section may have only one dominant visual idea.
- Each section may have only one job.
- Each section may have only one takeaway or action.

### Spatial Composition

- Use layout as a primary design tool.
- Favor unexpected composition when the context allows it.
- Use asymmetry, overlap, diagonal flow, grid-breaking moments, generous negative space, or controlled density when they strengthen the concept.
- Do not default to safe, evenly balanced blocks unless the product genuinely calls for restraint.
- Composition should make the interface memorable even before color and motion are added.

### Cards

- Do not use cards by default.
- Never use cards in the hero.
- Only use a card when the card itself is the interaction container.
- If removing the border, shadow, radius, or surface does not harm comprehension or interaction, it should not be a card.

### Typography

- Use no more than two typefaces.
- Use one accent color by default.
- On branded pages, the brand name or product name must appear at hero level.
- On branded pages, the headline must not overpower the brand.
- Copy must be readable within seconds.
- Supporting copy should usually be short and limited to one sentence.
- Choose typefaces that are beautiful, distinctive, and appropriate to the aesthetic direction.
- Avoid generic default-looking font choices unless brand, platform, or technical constraints require them.
- Avoid overused frontend defaults such as Arial, Roboto, Inter, and similar safe choices unless there is a strong system-level reason.
- Prefer a distinctive display face paired with a refined and readable body face.
- Typography must contribute to memorability, not merely legibility.

### Color and Surfaces

- Keep the number of colors low unless the aesthetic direction clearly requires more.
- Define tokens first.
- Use CSS variables or design tokens for consistency.
- Do not default to purple.
- Do not drift into dark mode unless the brief calls for it.
- Avoid clichéd color schemes, especially purple gradients on white backgrounds, unless there is a context-specific reason.
- Commit to a cohesive theme with dominant colors and sharp, intentional accents.
- Before adding chrome, solve problems with contrast, spacing, scale, cropping, and alignment.
- Atmosphere and depth should come from an intentional visual system, not from random decoration.

### Content

- Write in product language, not design commentary.
- Filler copy is forbidden.
- Do not repeat the same mood sentence across multiple sections.
- Do not expose prompt-like wording inside the UI.
- If removing 30 percent of the copy improves the result, remove more.
- Decorative text is strictly forbidden.
- Do not use words, phrases, or sentences as visual decoration.
- Never scatter keyword fragments, message excerpts, or thematic word lists just to make the layout feel designed.
- Text is allowed only when it is essential to meaning, usability, branding, navigation, or a clearly required message.
- The less the design depends on text and the more it communicates through composition, spacing, rhythm, scale, and imagery, the better.

### Motion

- Use 2 to 3 motions, not 10.
- One entrance sequence
- One scroll-linked or sticky behavior if needed
- One hover, reveal, or layout transition
- If a motion is purely decorative, remove it
- Focus on a few high-impact moments instead of scattering weak micro-interactions everywhere.
- One well-orchestrated page-load sequence with staggered reveals is often stronger than many unrelated animations.
- Prefer CSS-only motion for plain HTML implementations when practical.
- In React projects, use a motion library when available and appropriate.
- Motion should create hierarchy, delight, surprise, or feedback, never noise.

### Responsive Design

- The design must work cleanly on both desktop and mobile.
- The composition of the first viewport must survive on mobile.
- Tap targets and text contrast must remain strong even on top of imagery.
- Sticky or fixed headers should be treated as consuming layout budget in the initial viewport.

## Default Page Structures

## Landing Pages

Unless there is a clear reason not to, use the following sequence.

1. Hero

- Establish identity
- Establish the promise
- Present one clear action
- Present one dominant visual anchor

2. Support

- Present one concrete feature, offer, or proof point

3. Detail

- Explain the product, flow, mechanism, or narrative depth

4. Social proof

- Use only when it strengthens trust
- It is not required on every page

5. Final CTA

- Lead to one of the following: convert, start, book, contact, or continue

### Landing Page Hero Rules

- Use only one composition.
- Prefer a full-bleed hero or a full-canvas visual anchor by default.
- On branded landing pages, the hero should generally span edge to edge.
- Constrain only the inner text column and action column.
- Brand first, headline second, body third, CTA last.
- Keep the headline to roughly 2 to 3 lines on desktop.
- It must read at a glance on mobile.
- Keep the text column narrow and place it on a calm area of the image.
- Do not use hero cards.
- Do not default to stat strips, logo clouds, floating dashboards, or excessive pills.
- Do not use a split-screen hero unless one side is calm and visually unified.

### Landing Page Litmus Tests

- If the first viewport still works after removing the image, the image is too weak.
- If hiding the navigation makes the brand disappear, the hierarchy is too weak.
- If the first impression looks like a grid of UI devices, the composition is too weak.

## Apps and Dashboards

Default to restrained product UI.  
Aim for surfaces that feel calm, dense, readable, and operational.

### App Defaults

- Calm surface hierarchy
- Strong typography and spacing
- Few colors
- Dense but readable information
- Minimal chrome
- Use cards only when the card itself is the interaction unit

### Organizing Axes for App UI

- primary workspace
- navigation
- secondary context or inspector
- one clear accent for action or state

### Avoid in Apps

- Dashboard card mosaics
- Thick borders around every region
- Decorative gradients behind everyday UI
- Multiple competing accent colors
- Decorative icons that do not improve scanability
- Hero sections that feel like marketing pages unless explicitly requested

### App Copy Rules

- Prefer utilitarian copy over marketing copy
- Headings should communicate what the area is or what can be done there
- Prioritize orientation, state, action, and scanability
- The page should still make sense when reading only the headings, labels, and numbers

## Backgrounds and Visual Details

Backgrounds and visual details must build atmosphere and depth rather than defaulting to flat emptiness or arbitrary effects.

- Do not default to plain solid fills unless restraint is part of the concept.
- Use textures, transparency, gradients, patterns, grain, mesh effects, dramatic shadows, decorative borders, or custom cursor treatment only when they fit the aesthetic direction.
- Visual effects must feel integrated with the overall concept.
- Background treatment should support the interface, not compete with the content.
- Decorative detail must feel authored, not generated by habit.

## Image Rules

Images must do narrative work.  
Decorative texture alone is not enough.

- If the category benefits from imagery, use at least one strong, realistic image
- Prefer contextual photography over abstract gradients or fake 3D objects
- Choose or crop images so text sits on a stable tonal area
- Avoid images with too much signage, logos, or text
- Do not use images that already contain UI frames, cards, or panels
- If multiple moments are needed, use multiple images instead of a collage

When visual references are available:

- Extract spacing rhythm
- Extract scale relationships
- Extract contrast strategy
- Extract the balance between imagery and text
- Do not imitate blindly
- Adapt the visual language to the user’s product

When visual references are not available:

- Create a clear visual thesis and proceed
- Do not fall back to a generic card-grid SaaS layout

## Tool Use and Verification

If tools are available:

- Inspect the rendered result
- Verify hierarchy, spacing, contrast, and responsiveness
- Revise based on what is actually visible, not only on prior assumptions

Recommended flow:

1. Define the brief
2. Build the first pass
3. Render or inspect
4. Fix hierarchy, spacing, visual anchor, and interaction issues
5. Tighten the copy
6. Remove unnecessary cards, borders, shadows, and motion

## Output Contract

When using this skill, the model must produce work in the following order.

1. brief

- visual thesis
- content plan
- interaction thesis
- design system

2. structure

- section outline or screen layout
- explicit statement of the dominant visual idea for each section

3. build

- implement the UI

4. refine

- remove generic patterns
- tighten the copy
- reduce excessive chrome
- improve quality on both mobile and desktop

Do not output long design lectures.  
Do not explain every decision one by one.  
Be brief, decisive, and production-oriented.

## Failure Patterns That Must Be Rejected

If any one of the following appears, reject the draft and replace it.

- The first impression is a generic SaaS card grid
- The image is beautiful but the brand presence is weak
- The headline is strong but the action is unclear
- The image behind the text is visually noisy
- Multiple sections repeat the same message
- A carousel exists without a narrative role
- The app UI is built from stacked cards instead of layout
- There are multiple accent colors without a system-level reason
- There are more than two typefaces without a system-level reason
- The aesthetic direction is vague, generic, or inconsistent
- The typography feels default, safe, or interchangeable
- The palette feels timid, evenly distributed, or clichéd
- The layout feels predictable when the concept calls for a stronger point of view
- Visual effects exist without contributing to atmosphere or identity
- The interface looks like a generic AI-generated mockup rather than a designed product

## Final Checks

Before finishing, verify the following:

- Is the brand or product clear in the first screen
- Is there one strong visual anchor
- Can the page be understood by reading only the headlines
- Does each section have exactly one job
- Are cards truly necessary
- Does motion improve hierarchy or atmosphere
- Does the design still feel premium if decorative shadows are removed
- Does the mobile view still feel intentional rather than just a collapsed desktop layout
- Has all non-essential decorative text been removed

## Default Implementation Preferences

Unless the existing repository rules say otherwise, prefer the following:

- Use modern React patterns in React projects
- Prefer Tailwind or an existing design token system for fast iteration
- Define tokens before styling details spread across components
- Keep component count low in the initial draft
- Solve hierarchy through layout first, then polish

## Reminder to the Model

The intentionality of the model depends on the quality of the brief and the strength of the aesthetic commitment.  
Do not be generic.  
Do not pad the output.  
Do not hide weak structure behind decoration.  
Do not converge on overused defaults across generations.  
Make unexpected choices that still feel precise and context-aware.  
Choose a strong system early and commit to it.  
The model is capable of extraordinary creative work when it fully commits to a distinctive vision.

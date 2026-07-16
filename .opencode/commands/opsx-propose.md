---
description: Propose a new change - create it and generate all artifacts in one step
---

Propose a new change - create the change and generate all artifacts in one step.

I'll create a change with artifacts:

- intent.md (owner-confirmed meaning)
- proposal.md (what & why)
- design.md (how)
- tasks.md (implementation steps)

When ready to implement, run /opsx-apply

---

**Input**: The argument after `/opsx-propose` is the change name (kebab-case), OR a description of what the user wants to build.

Before starting, load `openspec-apply-readiness` via the `skill` tool and use it as the definition of apply-ready.

**Intent confirmation boundary**: Treat the request as evidence of intent, not as an implementation-ready specification. Every Change requires one explicit owner confirmation of the reconstructed intent before proposal, Specs, design, or tasks are authored.

**Steps**

1. **If no input provided, ask what they want to build**

   Use the **AskUserQuestion tool** (open-ended, no preset options) to ask:

   > "What change do you want to work on? Describe what you want to build or fix."

   From their description, derive a kebab-case name (e.g., "add user authentication" → `add-user-auth`).

   **IMPORTANT**: Do NOT proceed without understanding what the user wants to build.

2. **Create the change directory**

   ```bash
   openspec new change "<name>"
   ```

   This creates a scaffolded change at `openspec/changes/<name>/` with `.openspec.yaml`.

3. **Get the artifact build order**

   ```bash
   openspec status --change "<name>" --json
   ```

   Parse the JSON to get:
   - `applyRequires`: array of artifact IDs needed before implementation (e.g., `["tasks"]`)
   - `artifacts`: list of all artifacts with their status and dependencies

4. **Reconstruct and confirm intent**
   - Get the intent instructions:
     ```bash
     openspec instructions intent --change "<name>" --json
     ```
   - Inspect the relevant repository behavior, contracts, paths, and constraints before interpreting the request.
   - Build an intent candidate containing the actor, situation, problem, desired outcome, priority, request-term classifications, repository evidence, inferences, assumptions, falsification check, invariants, boundaries, and observable success.
   - Classify solution-shaped terms as `Required Outcome`, `Non-negotiable Constraint`, or `Candidate Means`.
   - Do not treat familiarity, common practice, searchable examples, or an existing implementation pattern as evidence that a candidate means fits this repository.
   - Present the complete candidate to the owner and ask them to confirm it, correct it, or stop.
   - If corrected, inspect any newly relevant evidence, revise the candidate, and ask again.
   - Only after explicit confirmation, create `intent.md` with exact `Intent-Status: CONFIRMED` and `Owner-Confirmation: CONFIRMED` markers and record the approved statement under `## Owner Confirmation`.
   - Do not create any downstream artifact while either status is unconfirmed or a decision remains unresolved that can change customer-visible behavior, contracts, architecture, security, data, dependencies, or scope.

5. **Create downstream artifacts in sequence until apply-ready**

   Use the **TodoWrite tool** to track progress through the artifacts.

   Loop through artifacts in dependency order (artifacts with no pending dependencies first):

   a. **For each artifact that is `ready` (dependencies satisfied)**:
   - Get instructions:
     ```bash
     openspec instructions <artifact-id> --change "<name>" --json
     ```
   - The instructions JSON includes:
     - `context`: Project background (constraints for you - do NOT include in output)
     - `rules`: Artifact-specific rules (constraints for you - do NOT include in output)
     - `template`: The structure to use for your output file
     - `instruction`: Schema-specific guidance for this artifact type
     - `outputPath`: Where to write the artifact
     - `dependencies`: Completed artifacts to read for context
   - Read any completed dependency files for context
   - Create the artifact file using `template` as the structure
   - Apply `context` and `rules` as constraints - but do NOT copy them into the file
   - Never select `proposal` or another downstream artifact until `intent.md` is confirmed
   - Show brief progress: "Created <artifact-id>"

   b. **Continue until all `applyRequires` artifacts are complete**
   - After creating each artifact, re-run `openspec status --change "<name>" --json`
   - Check if every artifact ID in `applyRequires` has `status: "done"` in the artifacts array
   - Stop when all `applyRequires` artifacts are done

   c. **If an artifact requires user input** (unclear context):
   - Use **AskUserQuestion tool** to clarify
   - Then continue with creation

   d. **Converge the shared apply-readiness contract**:
   - Run `openspec instructions apply --change "<name>" --json`
   - Read every returned `contextFiles` path
   - Evaluate AR-001 through AR-010 from `openspec-apply-readiness`
   - Fix every `NEEDS_FIXES` finding and ask for every required `NEEDS_DECISIONS` item
   - Do not declare the change apply-ready until the result is `READY`

6. **Show final status**
   ```bash
   openspec status --change "<name>"
   ```

**Output**

After completing all artifacts, summarize:

- Change name and location
- List of artifacts created with brief descriptions
- Confirmed intent path and owner-approved intent summary
- Apply-readiness result: `READY`
- What's ready: "All artifacts created! Ready for implementation."
- Prompt: "Run `/opsx-apply` to start implementing."

**Artifact Creation Guidelines**

- Follow the `instruction` field from `openspec instructions` for each artifact type
- The schema defines what each artifact should contain - follow it
- Read dependency artifacts for context before creating new ones
- Use `template` as the structure for your output file - fill in its sections
- **IMPORTANT**: `context` and `rules` are constraints for YOU, not content for the file
  - Do NOT copy `<context>`, `<rules>`, `<project_context>` blocks into the artifact
  - These guide what you write, but should never appear in the output

**Guardrails**

- Create ALL artifacts needed for implementation (as defined by schema's `apply.requires`)
- Always read dependency artifacts before creating a new one
- If context is critically unclear, ask the user - but prefer making reasonable decisions to keep momentum
- If a change with that name already exists, ask if user wants to continue it or create a new one
- Verify each artifact file exists after writing before proceeding to next
- Do not add local readiness gates or use expected file counts; use `openspec-apply-readiness`

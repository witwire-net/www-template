---
description: Orchestrates one OpenSpec change by loading the dedicated proposer workflow and coordinating designer, architects, and analyzer.
mode: subagent
model: openai/gpt-5.6-sol
reasoningEffort: 'xhigh'
temperature: 0.3
permission:
  edit:
    '*': allow
    'openspec/changes/**/*.wireframe.html': deny
    '*/openspec/changes/**/*.wireframe.html': deny
  'github_*': deny
  'github_get_*': allow
  'github_list_*': allow
  'github_search_*': allow
  github_issue_read: allow
  github_pull_request_read: allow
  github_run_secret_scanning: allow
  'agent-browser_*': allow
  serena_create_text_file: deny
  serena_insert_after_symbol: deny
  serena_insert_before_symbol: deny
  serena_execute_shell_command: deny
  serena_replace_content: deny
  serena_replace_symbol_body: deny
  serena_rename_symbol: deny
  serena_safe_delete_symbol: deny
  serena_write_memory: deny
  serena_edit_memory: deny
  serena_delete_memory: deny
  serena_rename_memory: deny
  serena_read_file: allow
  serena_search_for_pattern: allow
  webfetch: allow
  read_mcp_resource: allow
  skill: allow
  task:
    '*': deny
    'openspec/analyzer': allow
    'openspec/designer': allow
    'openspec/frontend/architect': allow
    'openspec/backend/architect': allow
  read:
    '*': allow
    '*.env': deny
    '*.env.*': deny
    '*.env.example': allow
  glob: allow
  grep: allow
  list: allow
  lsp: allow
  bash:
    '*': allow
    'git add*': deny
    'git commit*': deny
    'git push*': deny
    'git -C *': deny
    'git reset*': deny
    'git clean*': deny
    'git checkout*': deny
    'git switch*': deny
    'git restore*': deny
    'git branch -d*': deny
    'git branch -D*': deny
    'git worktree remove*': deny
    'git worktree prune*': deny
    'git rebase*': deny
    'git merge*': deny
    'pnpm deploy*': deny
    'pnpm run deploy*': deny
    'pnpm release:*': deny
    'pnpm run release:*': deny
    'pnpm migrate:create*': deny
    'pnpm migrate:up*': deny
    'pnpm migrate:down*': deny
    'pnpm migrate:force*': deny
    'pnpm exec wrangler deploy*': deny
    'npx wrangler deploy*': deny
    'wrangler deploy*': deny
    'gh *': deny
    'rm *': deny
---

# OpenSpec proposer

You are the `openspec/proposer` subagent.

For every invocation, first load `openspec-proposer-workflow` via `skill` and
execute it as the sole operating contract for proposal work. Do not begin
artifact work before loading it.

Do not substitute the upstream `openspec-propose` workflow or the
`/opsx-propose` command, and do not duplicate workflow instructions in this
agent definition.

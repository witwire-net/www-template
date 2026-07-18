import { spawnSync } from 'node:child_process';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

import { collectActiveChangeArtifacts } from '#openspec/change-artifacts.mjs';

const SCRIPT_DIRECTORY = path.dirname(fileURLToPath(import.meta.url));
const REPOSITORY_ROOT = path.resolve(SCRIPT_DIRECTORY, '..', '..');
const GENERATOR_PATH = path.join(
  REPOSITORY_ROOT,
  '.opencode',
  'skills',
  'wireframe',
  'scripts',
  'generate-preview.mjs'
);
const JSON_SUFFIX = '.wireframe.json';

// OpenSpec は active Change の収集だけを所有し、preview の生成規則は wireframe skill に委譲します。
const sourcePaths = collectActiveChangeArtifacts(process.cwd(), (_absolutePath, fileName) =>
  fileName.endsWith(JSON_SUFFIX)
);

if (sourcePaths.length > 0) {
  const result = spawnSync(process.execPath, [GENERATOR_PATH, '--check', ...sourcePaths], {
    cwd: process.cwd(),
    encoding: 'utf8',
  });

  if (result.stdout) process.stdout.write(result.stdout);
  if (result.stderr) process.stderr.write(result.stderr);
  if (result.status !== 0) process.exitCode = result.status ?? 1;
}

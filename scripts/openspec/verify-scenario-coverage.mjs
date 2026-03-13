import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const SCENARIO_ID_PATTERN = /^[\dA-Z]+(?:-[\dA-Z]+)*-S\d{3,}$/;
const SCENARIO_REF_PATTERN = /\[(?<id>[\dA-Z]+(?:-[\dA-Z]+)*-S\d{3,})]/g;

const REPO_ROOT = process.cwd();

/**
 * @param {string} absDir
 * @param {(absPath: string) => boolean} includeFile
 * @param {(name: string) => boolean} ignoreDir
 */
function collectFiles(absDir, includeFile, ignoreDir) {
  /** @type {string[]} */
  const out = [];

  /** @type {string[]} */
  const stack = [absDir];
  while (stack.length > 0) {
    const current = stack.pop();
    if (!current) continue;
    let entries;
    try {
      entries = readdirSync(current, { withFileTypes: true });
    } catch {
      continue;
    }

    for (const ent of entries) {
      const abs = path.join(current, ent.name);
      if (ent.isDirectory()) {
        if (ignoreDir(ent.name)) continue;
        stack.push(abs);
        continue;
      }
      if (ent.isFile() && includeFile(abs)) {
        out.push(abs);
      }
    }
  }

  return out;
}

/**
 * @param {string} absPath
 */
function readText(absPath) {
  return readFileSync(absPath, 'utf8');
}

/**
 * @typedef {'added'|'modified'|'removed'|'renamed'|null} DeltaSection
 */

/**
 * @typedef {{
 *   id: string;
 *   absPath: string;
 *   relPath: string;
 *   line: number;
 *   manual: boolean;
 *   included: boolean;
 * }} Scenario
 */

/**
 * @param {string} absPath
 * @returns {{ scenarios: Scenario[]; errors: string[] }}
 */
function parseSpecFile(absPath) {
  const relPath = path.relative(REPO_ROOT, absPath);
  const lines = readText(absPath).split(/\r?\n/);

  /** @type {Scenario[]} */
  const scenarios = [];
  /** @type {string[]} */
  const errors = [];

  /** @type {DeltaSection} */
  let deltaSection = null;
  let seenDeltaSections = false;

  const setDeltaSectionFromLine = (line) => {
    const m = /^##\s+(added|modified|removed|renamed)\s+requirements\b/i.exec(line);
    if (!m) return;
    seenDeltaSections = true;
    const k = m[1];
    if (k === 'added') deltaSection = 'added';
    else if (k === 'modified') deltaSection = 'modified';
    else if (k === 'removed') deltaSection = 'removed';
    else if (k === 'renamed') deltaSection = 'renamed';
  };

  const parseTags = (tagsRaw) =>
    tagsRaw
      .split(',')
      .map((t) => t.trim().toLowerCase())
      .filter(Boolean);

  const isScenarioHeading = (line) => /^#{4}\s+Scenario:\s+/.test(line);

  const extractScenarioId = (line) => {
    const idMatch = /^#{4}\s+Scenario:\s+.*\(([^)]+)\)\s*$/.exec(line);
    if (!idMatch) {
      return { id: null, error: 'Scenario heading must end with a stable ID in parentheses (e.g. (USER-MGMT-S001)).' };
    }
    const id = idMatch[1].trim();
    if (!SCENARIO_ID_PATTERN.test(id)) {
      return {
        id: null,
        error: `Invalid Scenario ID '${id}'. Expected pattern: ${String(SCENARIO_ID_PATTERN)}`,
      };
    }
    return { id, error: null };
  };

  const isHeading = (line) => /^#{1,4}\s+/.test(line);

  const isManualScenario = (startLineIndex) => {
    const scan = lines.slice(startLineIndex + 1, Math.min(lines.length, startLineIndex + 21));
    for (const l of scan) {
      if (isHeading(l)) break;
      const tagLine =
        /^tags:\s*(.+)\s*$/i.exec(l) ?? /^<!--\s*tags:\s*(.+?)\s*-->\s*$/i.exec(l);
      if (!tagLine) continue;
      const tags = parseTags(tagLine[1]);
      return tags.includes('manual');
    }
    return false;
  };

  for (const [i, line] of lines.entries()) {
    setDeltaSectionFromLine(line);

    if (!isScenarioHeading(line)) continue;

    const extracted = extractScenarioId(line);
    if (extracted.error || !extracted.id) {
      errors.push(`${relPath}:${i + 1}: ${extracted.error ?? 'Invalid Scenario heading.'}`);
      continue;
    }

    const id = extracted.id;

    const manual = isManualScenario(i);

    const inDelta = seenDeltaSections;
    const included = !inDelta || deltaSection !== 'removed';
    scenarios.push({ id, absPath, relPath, line: i + 1, manual, included });
  }

  return { scenarios, errors };
}

function ensureDirExists(relDir) {
  const abs = path.join(REPO_ROOT, relDir);
  try {
    return statSync(abs).isDirectory();
  } catch {
    return false;
  }
}

const writeOut = (line) => {
  process.stdout.write(`${line}\n`);
};

const writeErr = (line) => {
  process.stderr.write(`${line}\n`);
};

function getSpecFiles() {
  if (!ensureDirExists('openspec/specs')) return [];
  return collectFiles(
    path.join(REPO_ROOT, 'openspec/specs'),
    (abs) => abs.endsWith('spec.md'),
    (dirName) => dirName === '.git' || dirName === 'archive'
  );
}

/**
 * @param {string[]} specFiles
 */
function loadScenarios(specFiles) {
  /** @type {Scenario[]} */
  const scenarios = [];
  /** @type {string[]} */
  const parseErrors = [];

  for (const f of specFiles) {
    const { scenarios: s, errors } = parseSpecFile(f);
    scenarios.push(...s.filter((x) => x.included));
    parseErrors.push(...errors);
  }

  return { scenarios, parseErrors };
}

/**
 * @param {Scenario[]} scenarios
 */
function indexScenariosById(scenarios) {
  /** @type {Map<string, Scenario[]>} */
  const byId = new Map();
  for (const s of scenarios) {
    const list = byId.get(s.id) ?? [];
    list.push(s);
    byId.set(s.id, list);
  }
  return byId;
}

/**
 * @param {Map<string, Scenario[]>} byId
 */
function getDuplicateScenarioErrors(byId) {
  /** @type {string[]} */
  const duplicateErrors = [];
  for (const [id, list] of byId.entries()) {
    if (list.length <= 1) continue;
    const where = list
      .map((x) => `${x.relPath}:${x.line}${x.manual ? ' (manual)' : ''}`)
      .sort()
      .join(', ');
    duplicateErrors.push(`Duplicate Scenario ID '${id}': ${where}`);
  }
  return duplicateErrors;
}

function getTestFiles() {
  /** @type {string[]} */
  const testFiles = [];
  const testFileMatcher = (abs) => /\.(test|spec)\.(ts|tsx)$/.test(abs);
  const ignoreDir = (name) =>
    name === 'node_modules' ||
    name === 'dist' ||
    name === 'build' ||
    name === '.git' ||
    name === '.wrangler' ||
    name === 'coverage' ||
    name === 'playwright-report' ||
    name === 'test-results';

  if (ensureDirExists('packages')) {
    testFiles.push(...collectFiles(path.join(REPO_ROOT, 'packages'), testFileMatcher, ignoreDir));
  }
  if (ensureDirExists('tests')) {
    testFiles.push(...collectFiles(path.join(REPO_ROOT, 'tests'), testFileMatcher, ignoreDir));
  }

  return testFiles;
}

/**
 * @param {string[]} testFiles
 */
function collectTestScenarioReferences(testFiles) {
  /** @type {Map<string, Set<string>>} */
  const referencedIn = new Map();
  for (const f of testFiles) {
    const rel = path.relative(REPO_ROOT, f);
    const content = readText(f);
    for (const m of content.matchAll(SCENARIO_REF_PATTERN)) {
      const id = m.groups?.id;
      if (!id) continue;
      const set = referencedIn.get(id) ?? new Set();
      set.add(rel);
      referencedIn.set(id, set);
    }
  }
  return referencedIn;
}

/**
 * @param {Map<string, Scenario[]>} byId
 * @param {Scenario[]} scenarios
 * @param {Map<string, Set<string>>} referencedIn
 */
function computeCoverage(byId, scenarios, referencedIn) {
  const specIdsAll = new Set(byId.keys());
  const requiredIds = scenarios.filter((s) => !s.manual).map((s) => s.id);
  const missing = requiredIds.filter((id) => !referencedIn.has(id));
  const orphans = [...referencedIn.keys()].filter((id) => !specIdsAll.has(id));
  return { missing, orphans };
}

/**
 * @param {{
 *  parseErrors: string[];
 *  duplicateErrors: string[];
 *  missing: string[];
 *  orphans: string[];
 *  byId: Map<string, Scenario[]>;
 *  referencedIn: Map<string, Set<string>>;
 * }} result
 */
function report(result) {
  const { parseErrors, duplicateErrors, missing, orphans, byId, referencedIn } = result;
  const ok =
    parseErrors.length === 0 &&
    duplicateErrors.length === 0 &&
    missing.length === 0 &&
    orphans.length === 0;

  if (ok) {
    writeOut('OpenSpec scenario coverage: OK');
    return;
  }

  writeErr('OpenSpec scenario coverage: FAILED');

  if (parseErrors.length > 0) {
    writeErr(`\nSpec parse errors (${parseErrors.length}):`);
    for (const e of parseErrors.sort()) writeErr(`- ${e}`);
  }

  if (duplicateErrors.length > 0) {
    writeErr(`\nDuplicate Scenario IDs (${duplicateErrors.length}):`);
    for (const e of duplicateErrors.sort()) writeErr(`- ${e}`);
  }

  if (missing.length > 0) {
    writeErr(`\nMissing test references for Scenario IDs (${missing.length}):`);
    for (const id of missing.sort()) {
      const meta = byId.get(id)?.[0];
      const where = meta ? `${meta.relPath}:${meta.line}` : '(unknown)';
      writeErr(`- ${id} (from ${where})`);
    }
    writeErr(
      "\nHint: add the ID in the test title like: it('[USER-MGMT-S001] ...', ...) or test('[USER-MGMT-S001] ...', ...)"
    );
  }

  if (orphans.length > 0) {
    writeErr(`\nScenario IDs referenced in tests but not found in specs (${orphans.length}):`);
    for (const id of orphans.sort()) {
      const files = [...(referencedIn.get(id) ?? new Set())].sort().join(', ');
      writeErr(`- ${id} (referenced in: ${files})`);
    }
  }

  process.exitCode = 1;
}

function main() {
  const specFiles = getSpecFiles();
  const { scenarios, parseErrors } = loadScenarios(specFiles);
  const byId = indexScenariosById(scenarios);
  const duplicateErrors = getDuplicateScenarioErrors(byId);
  const testFiles = getTestFiles();
  const referencedIn = collectTestScenarioReferences(testFiles);
  const { missing, orphans } = computeCoverage(byId, scenarios, referencedIn);

  report({ parseErrors, duplicateErrors, missing, orphans, byId, referencedIn });
}

main();

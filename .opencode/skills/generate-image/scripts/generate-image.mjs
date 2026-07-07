#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { access, copyFile, mkdir, readFile, readdir, stat } from 'node:fs/promises';
import { constants as fsConstants } from 'node:fs';
import { dirname, extname, isAbsolute, relative, resolve, sep } from 'node:path';
import { homedir } from 'node:os';

const VERSION = '0.1.0';
const DEFAULT_TIMEOUT_MS = 10 * 60 * 1000;
const DEFAULT_OUTPUT_DIR = resolve(homedir(), 'Pictures', 'codex-images');
const IMAGE_EXTENSIONS = new Set(['.png', '.jpg', '.jpeg', '.webp']);
const VALID_QUALITIES = new Set(['low', 'medium', 'high', 'auto']);
const REPEATABLE_ARGS = new Set(['image', 'imageRole', 'constraint']);
const GENERATED_IMAGE_LOOKBACK_MS = 30 * 1000;

const TEMPLATES = {
  general: {
    firstLine: 'Create a polished raster image for the requested use case.',
    purpose: 'General-purpose visual generation.',
    subject: "Follow the user's primary request exactly and avoid inventing unrelated elements.",
    composition: 'Use a clear focal point, intentional spacing, and a practical visual hierarchy.',
    style: 'Choose a coherent visual language that supports the requested use case.',
    constraints: [
      'No watermark.',
      'No fake logos or trademarks unless explicitly requested.',
      'No placeholder image.',
    ],
  },
  'ui-mockup': {
    firstLine: 'Create a realistic web app UI mockup for the requested product or workflow.',
    purpose: 'Product screenshot-style visual for reviewing UI direction and product realism.',
    subject:
      'A polished product UI screen with practical components, readable hierarchy, and implementable spacing.',
    composition:
      'Prioritize information architecture, navigation roles, primary actions, and screen-level hierarchy.',
    style:
      'Realistic shippable product UI, readable typography, coherent color system, practical component hierarchy, not concept art.',
    constraints: [
      'No fake logos or trademarks.',
      'No watermark.',
      'No decorative badges unless requested.',
      'No unreadable dense microtext.',
      'No random analytics charts unless requested.',
      'No authentication screen unless requested.',
    ],
    wireframePolicy: [
      'Use the wireframe as layout and information architecture guidance only.',
      'Preserve the major sections, hierarchy, content groups, navigation roles, and primary actions.',
      "Do not copy the wireframe's low-fidelity visual style.",
      'Do not render gray placeholder boxes, generic outlines, equal-weight rectangles, or scaffold labels as final UI styling.',
    ],
  },
  'product-mockup': {
    firstLine:
      'Create a production-quality product mockup visual for the requested product or service.',
    purpose:
      'Marketing or presentation image that makes the product/service easy to understand and attractive.',
    subject:
      'The product, packaging, device, service screen, or branded object described by the user.',
    composition:
      'Use clean silhouette, controlled focal point, usable whitespace, and a premium presentation layout.',
    style:
      'Polished product photography or product-render visual with controlled lighting and realistic material detail.',
    constraints: [
      'No watermark.',
      'No unrelated props.',
      'No fake logos or trademarks unless explicitly requested.',
      'No invented label text.',
    ],
  },
  'landing-hero': {
    firstLine: 'Create a landing-page hero visual for the requested product, service, or campaign.',
    purpose: 'Above-the-fold website visual that supports headline, CTA, and product positioning.',
    subject:
      'The central product or concept described by the user, simplified for fast comprehension.',
    composition:
      'Wide composition, clear hierarchy, intentional negative space for page copy, and a restrained visual cluster.',
    style: 'Web-ready hero image with polished but not overdecorated product visuals.',
    constraints: [
      'No watermark.',
      'No fake logos.',
      'No decorative badges unless requested.',
      'No clutter.',
      'No unreadable UI text.',
    ],
  },
  'ad-creative': {
    firstLine: 'Create a polished advertising creative for the requested audience and channel.',
    purpose: 'Campaign image that communicates the offer, mood, and audience fit quickly.',
    subject: 'The campaign subject and any exact text supplied by the user.',
    composition:
      'Use strong hierarchy, a clear focal area, and readable space for text if text is requested.',
    style:
      'Tasteful campaign photography, editorial design, or poster-like layout according to the request.',
    constraints: [
      'No watermark.',
      'No unrelated logos.',
      'No duplicate text.',
      'No extra copy beyond requested text.',
    ],
  },
  infographic: {
    firstLine: 'Create a clean infographic or explanatory diagram for the requested topic.',
    purpose:
      'Explain a process, comparison, or concept with readable hierarchy and scan-friendly structure.',
    subject:
      "The required diagram entities, steps, labels, arrows, or sections from the user's request.",
    composition: 'Use a clear flow, large labels, generous whitespace, and a strong reading order.',
    style:
      'Minimal editorial infographic, high contrast, readable typography, no decorative clutter.',
    constraints: [
      'No watermark.',
      'No extra labels.',
      'No unreadable microtext.',
      'No random statistics.',
    ],
  },
  'product-shot': {
    firstLine: 'Create a premium product-shot image for the requested product.',
    purpose: 'Product photo suitable for catalog, e-commerce, landing page, or pitch material.',
    subject: 'A single clear product subject unless the user requests a set or bundle.',
    composition:
      'Clean silhouette, controlled camera angle, natural contact shadow, and generous padding.',
    style:
      'Premium product photography with realistic materials, label clarity, and controlled studio lighting.',
    constraints: [
      'No watermark.',
      'No invented brand marks.',
      'No extra props unless requested.',
      'Do not redesign labels unless requested.',
    ],
  },
  'transparent-cutout': {
    firstLine: 'Create a clean cutout-ready raster image for background removal.',
    purpose:
      'Generate a subject on a removable flat chroma-key background for later alpha extraction.',
    subject: 'The requested subject, isolated with crisp edges and generous padding.',
    composition: 'Centered subject, full silhouette visible, no cropped important edges.',
    style: 'Clean product or object rendering with a perfectly flat solid background.',
    constraints: [
      'Use a perfectly flat solid #00ff00 background unless the subject is green; then use #ff00ff.',
      'No shadows, gradients, texture, reflections, floor plane, or lighting variation in the background.',
      'No watermark.',
      'Do not use the key color inside the subject.',
    ],
  },
  'image-edit': {
    firstLine: 'Edit the input image according to the requested change.',
    purpose: 'Precise image edit with explicit preservation of unchanged areas.',
    subject: 'The provided input image and the user-specified edit target.',
    composition:
      'Preserve camera angle, framing, scale, and scene geometry unless the user explicitly requests otherwise.',
    style: "Match the original image's lighting, texture, perspective, and physical realism.",
    constraints: [
      'No watermark.',
      'No redesign unless requested.',
      'No extra objects.',
      'No identity or label drift.',
    ],
    editMode: true,
  },
  'reference-composite': {
    firstLine: 'Create a composite or reference-guided image using the supplied image roles.',
    purpose:
      'Use multiple references with explicit roles while preserving the requested source details.',
    subject: 'The final image described by the user, assembled from the role-labeled references.',
    composition: 'Match scale, lighting, perspective, and layout according to the reference roles.',
    style:
      'Coherent final image that uses references deliberately without copying unrelated details.',
    constraints: [
      'No watermark.',
      'No unintended logo copying.',
      'Do not redesign preserved subjects.',
      'Do not invent new label text.',
    ],
  },
};

/**
 * CLI の入口です。
 *
 * 引数を解析し、必要な入力を読み、Codex に渡す成果物仕様書型 prompt を作ります。
 * `--dry-run` が指定された場合は副作用を持たず、生成される prompt と command だけを表示します。
 * 通常実行では output directory を作成し、Codex CLI を起動し、指定された出力画像の存在を確認します。
 */
async function main() {
  const args = parseArgs(process.argv.slice(2));

  if (args.help) {
    process.stdout.write(helpText());
    return;
  }

  if (args.version) {
    process.stdout.write(`${VERSION}\n`);
    return;
  }

  const request = normalizeRequest(args);
  const wireframeSummary = request.wireframePath
    ? summarizeWireframe(await loadWireframe(request.wireframePath))
    : null;
  const prompt = buildCodexPrompt(request, wireframeSummary);
  const command = buildCodexCommand(request);

  if (request.dryRun) {
    process.stdout.write(renderDryRun(command, prompt, request));
    return;
  }

  await mkdir(request.outputDir, { recursive: true });
  const startedAtMs = Date.now();
  const result = await runCodex(command, prompt, request.timeoutMs);

  if (result.exitCode !== 0) {
    throw new Error(
      [
        'Codex image generation failed.',
        `exitCode: ${result.exitCode}`,
        result.stderr ? `stderr:\n${tail(result.stderr)}` : null,
        result.stdout ? `stdout:\n${tail(result.stdout)}` : null,
      ]
        .filter(Boolean)
        .join('\n\n')
    );
  }

  const output = await resolveOutputImage(request.outputPath, result, startedAtMs);
  process.stdout.write(`${output}\n`);
}

/**
 * CLI 引数を小さな独自 parser で解析します。
 *
 * 外部依存を増やさず、`--key value`、`--key=value`、boolean flag、繰り返し指定を扱います。
 * `--image`、`--image-role`、`--constraint` は複数回指定できるため配列として保持します。
 */
function parseArgs(argv) {
  const args = { image: [], imageRole: [], constraint: [] };

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];

    if (!token.startsWith('--')) {
      throw new Error(`Unexpected positional argument: ${token}`);
    }

    const eqIndex = token.indexOf('=');
    const rawKey = eqIndex === -1 ? token.slice(2) : token.slice(2, eqIndex);
    const key = camelCase(rawKey);
    const inlineValue = eqIndex === -1 ? null : token.slice(eqIndex + 1);

    if (['dryRun', 'help', 'version'].includes(key)) {
      args[key] = true;
      continue;
    }

    const value = inlineValue ?? argv[index + 1];
    if (value === undefined || value.startsWith('--')) {
      throw new Error(`Missing value for --${rawKey}`);
    }
    if (inlineValue === null) {
      index += 1;
    }

    if (REPEATABLE_ARGS.has(key)) {
      args[key].push(value);
    } else {
      args[key] = value;
    }
  }

  return args;
}

/**
 * 生の CLI 引数を実行に使う正規化済み request に変換します。
 *
 * ここで template、quality、size、出力 path、画像 path を検証します。
 * 画像生成前に失敗させることで、Codex に曖昧な指示を渡さないようにします。
 */
function normalizeRequest(args) {
  const templateName = args.template ?? 'general';
  const template = TEMPLATES[templateName];
  if (!template) {
    throw new Error(
      `Unknown template: ${templateName}. Available: ${Object.keys(TEMPLATES).join(', ')}`
    );
  }

  const prompt = requiredString(args.prompt, 'prompt');
  const quality = args.quality ?? 'medium';
  if (!VALID_QUALITIES.has(quality)) {
    throw new Error(`Invalid quality: ${quality}. Use low, medium, high, or auto.`);
  }

  const size = args.size ?? 'auto';
  validateSize(size);

  const outputPath = resolveOutputPath(args.out, prompt);
  const outputDir = dirname(outputPath);
  const timeoutMs = parsePositiveInteger(args.timeoutMs ?? DEFAULT_TIMEOUT_MS, 'timeoutMs');
  const images = args.image.map((value) => resolve(value));
  const imageRoles = args.imageRole;

  if (imageRoles.length > images.length) {
    throw new Error('--image-role cannot be specified more times than --image.');
  }

  if (template.editMode && images.length === 0) {
    throw new Error('The image-edit template requires at least one --image input.');
  }

  const form = normalizeForm(args);

  return {
    templateName,
    template,
    prompt,
    form,
    quality,
    size,
    outputPath,
    outputDir,
    timeoutMs,
    dryRun: Boolean(args.dryRun),
    iterationTarget: args.iterationTarget ?? defaultIterationTarget(templateName),
    wireframePath: args.wireframe ? resolve(args.wireframe) : null,
    images,
    imageRoles,
    codexBin: process.env.GENERATE_IMAGE_CODEX_BIN || process.env.CODEX_BIN || 'codex',
  };
}

/**
 * CLI から渡された成果物仕様書フォームの欄を正規化します。
 *
 * 各欄は Codex に渡す prompt の固定 section に対応します。
 * 未指定欄は template 既定値や共通 safety rule で補完し、利用者が巨大な自由文 prompt を毎回書かなくて済むようにします。
 */
function normalizeForm(args) {
  return {
    purpose: optionalString(args.purpose),
    canvas: optionalString(args.canvas),
    subject: optionalString(args.subject),
    composition: optionalString(args.composition),
    style: optionalString(args.style),
    text: optionalString(args.text),
    typography: optionalString(args.typography),
    details: optionalString(args.details),
    preserve: optionalString(args.preserve),
    changeOnly: optionalString(args.changeOnly),
    physicalRealism: optionalString(args.physicalRealism),
    constraints: [
      ...splitBlock(args.constraints),
      ...args.constraint.map((value) => value.trim()).filter(Boolean),
    ],
  };
}

/**
 * wireframe file を読み込みます。
 *
 * `.wireframe.json` はそのまま JSON として解析します。
 * `.wireframe.html` は wireframe skill の preview が含む `const WIREFRAME_DATA = ...;` を抽出します。
 */
async function loadWireframe(filePath) {
  const text = await readFile(filePath, 'utf8');
  const extension = extname(filePath).toLowerCase();

  if (extension === '.json') {
    return JSON.parse(text);
  }

  if (extension === '.html' || extension === '.htm') {
    const match = text.match(/const\s+WIREFRAME_DATA\s*=\s*([\s\S]*?);\s*(?:\n|$)/);
    if (!match) {
      throw new Error(`Unable to find WIREFRAME_DATA in ${filePath}`);
    }
    return JSON.parse(match[1]);
  }

  throw new Error(`Unsupported wireframe file extension: ${extension}. Use .json or .html.`);
}

/**
 * wireframe JSON を GPT-Image-2 向けの短い UI brief に変換します。
 *
 * 変換は AI 要約ではなく deterministic な tree traversal です。
 * 低忠実度 renderer の見た目ではなく、画面名、viewport、layout、役割、主要 action を抽出します。
 */
function summarizeWireframe(wireframe) {
  const root = wireframe.root ?? {};
  const viewport = wireframe.viewport ?? {};
  const bullets = [];
  const actions = [];
  const tables = [];
  const headings = [];

  walkWireframe(root, (node, depth) => {
    if (depth <= 4 && bullets.length < 36) {
      bullets.push(`${'  '.repeat(depth)}- ${describeNode(node)}`);
    }

    if (node.type === 'button' || node.type === 'link') {
      actions.push(
        `${node.name ?? 'Unnamed action'}${node.type === 'button' ? ' (primary/action button)' : ' (secondary/link action)'}`
      );
    }

    if (node.type === 'table') {
      const columns = Array.isArray(node.columns)
        ? node.columns.map((column) => column.name).filter(Boolean)
        : [];
      tables.push(
        `${node.name ?? 'Table'}${columns.length > 0 ? `: columns ${columns.join(', ')}` : ''}`
      );
    }

    if (node.type === 'text' && ['display', 'heading'].includes(node.variant)) {
      headings.push(
        `${node.name ?? 'Untitled heading'}${node.variant ? ` (${node.variant})` : ''}`
      );
    }
  });

  return {
    screenName: wireframe.name ?? 'Unnamed screen',
    viewport: describeViewport(viewport),
    layout: describeRootLayout(root),
    bullets,
    actions: unique(actions).slice(0, 12),
    tables: unique(tables).slice(0, 8),
    headings: unique(headings).slice(0, 8),
  };
}

/**
 * Codex に渡す最終 prompt を生成します。
 *
 * 全 template 共通の安全 guardrail と、template 固有の成果物仕様を結合します。
 * wireframe がある場合だけ、UI brief と low-fidelity leakage を避ける指示を追加します。
 */
function buildCodexPrompt(request, wireframeSummary) {
  const referenceImages = renderImageRoles(request.images, request.imageRoles);
  const wireframeBlock = wireframeSummary
    ? renderWireframeBlock(request.template, wireframeSummary)
    : null;

  return [
    "You are being invoked by OpenCode's generate-image skill to create exactly one raster image.",
    '',
    "Use Codex's built-in image generation tool only.",
    'The tool may be named image_gen.imagegen, image_gen, or image generation.',
    'Do not use external image APIs.',
    'Do not use curl.',
    'Do not use Python SDKs.',
    'Do not create SVG, HTML, CSS, canvas, or placeholder images.',
    'Generate exactly one real raster image.',
    '',
    `TARGET_IMAGE_PATH: ${request.outputPath}`,
    '',
    request.prompt,
    '',
    'Purpose:',
    request.form.purpose ?? request.template.purpose,
    '',
    'Canvas:',
    ...renderCanvasLines(request),
    '',
    'Subject:',
    request.form.subject ?? request.template.subject,
    '',
    'Composition:',
    request.form.composition ?? request.template.composition,
    '',
    wireframeBlock,
    referenceImages,
    'Style:',
    request.form.style ?? request.template.style,
    '',
    renderEditBlock(request),
    'Text:',
    request.form.text ??
      'Use exact quoted text only when the user provided exact text. Otherwise use minimal neutral UI/content placeholders and avoid fake paragraphs.',
    '',
    renderOptionalSection('Typography', request.form.typography),
    renderOptionalSection('Details', request.form.details),
    request.template.editMode ? null : renderOptionalSection('Preserve', request.form.preserve),
    'Quality target:',
    `${request.quality}. Use low for exploration, medium for normal production direction, and high for text-heavy/final/detail-sensitive output.`,
    '',
    'Constraints:',
    ...renderConstraintLines(request),
    '- No duplicate text.',
    '- No misspellings for requested exact text.',
    '- No extra text beyond the requested content.',
    '',
    'Iteration target:',
    request.iterationTarget,
    '',
    'Required final behavior:',
    '1. Use the built-in image generation tool to generate the image.',
    '2. Do not use shell commands, filesystem helpers, or sandbox copy/move operations to place the image at TARGET_IMAGE_PATH.',
    '3. If the image generation tool exposes a generated artifact path, return only that artifact path. Otherwise return only a concise completion message.',
  ]
    .filter((line) => line !== null && line !== undefined)
    .join('\n');
}

/**
 * Canvas section を固定順で作ります。
 *
 * `--canvas` は媒体、余白、crop、safe area などの成果物仕様として扱い、`--size` は Codex built-in tool に直接渡す API 値ではなく出力意図として併記します。
 */
function renderCanvasLines(request) {
  const lines = [];
  if (request.form.canvas) {
    lines.push(request.form.canvas);
  }
  lines.push(
    `Preferred canvas: ${request.size}. Treat this as output intent if the built-in tool does not expose explicit sizing.`
  );
  lines.push('Keep safe margins and avoid important details at the extreme edges.');
  return lines;
}

/**
 * 任意 section を、未指定なら完全に省略します。
 */
function renderOptionalSection(title, body) {
  if (!body) {
    return null;
  }
  return [`${title}:`, body, ''].join('\n');
}

/**
 * template 固有の制約と CLI フォームから渡された制約をまとめます。
 */
function renderConstraintLines(request) {
  return [...request.template.constraints, ...request.form.constraints].map(
    (constraint) => `- ${constraint}`
  );
}

/**
 * Codex CLI の command line を構築します。
 *
 * `--image` が指定された場合は Codex CLI に reference image として渡します。
 * 実行ディレクトリは出力先 directory に固定し、Codex が workspace-write sandbox 内で画像を保存できるようにします。
 */
function buildCodexCommand(request) {
  const args = [
    'exec',
    '--skip-git-repo-check',
    '--ephemeral',
    '--sandbox',
    'workspace-write',
    '--json',
    '--cd',
    request.outputDir,
  ];

  for (const imagePath of request.images) {
    args.push('--image', imagePath);
  }

  args.push('-');
  return { bin: request.codexBin, args, cwd: request.outputDir };
}

/**
 * Codex CLI を起動し、stdin に prompt を渡します。
 *
 * stdout/stderr は診断用に保存します。timeout 到達時は child process を終了し、呼び出し元へ明示的な error を返します。
 */
async function runCodex(command, prompt, timeoutMs) {
  await mkdir(command.cwd, { recursive: true });

  return await new Promise((resolvePromise, rejectPromise) => {
    const child = spawn(command.bin, command.args, {
      cwd: command.cwd,
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';
    const generatedImagePaths = new Set();
    let settled = false;
    const timeout = setTimeout(() => {
      child.kill('SIGTERM');
      rejectOnce(new Error(`Codex image generation timed out after ${timeoutMs}ms.`));
    }, timeoutMs);

    const rejectOnce = (error) => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timeout);
      rejectPromise(error);
    };

    child.stdout.on('data', (chunk) => {
      const text = chunk.toString('utf8');
      collectGeneratedImagePathsFromText(text, generatedImagePaths);
      stdout = limitBuffer(stdout + text);
    });

    child.stderr.on('data', (chunk) => {
      const text = chunk.toString('utf8');
      collectGeneratedImagePathsFromText(text, generatedImagePaths);
      stderr = limitBuffer(stderr + text);
    });

    child.on('error', (error) => {
      rejectOnce(new Error(`Unable to start Codex command: ${error.message}`));
    });

    child.on('close', (exitCode, signal) => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timeout);
      collectGeneratedImagePathsFromText(stdout, generatedImagePaths);
      collectGeneratedImagePathsFromText(stderr, generatedImagePaths);
      resolvePromise({
        exitCode,
        signal,
        stdout,
        stderr,
        generatedImagePaths: [...generatedImagePaths],
      });
    });

    child.stdin.end(prompt);
  });
}

/**
 * 出力画像の存在を確認します。
 *
 * まず exact path を確認します。
 * Codex の filesystem helper が sandbox 制約で copy に失敗する環境では、Codex 側の生成 artifact を Node 側で回収して配置します。
 */
async function resolveOutputImage(outputPath, codexResult, startedAtMs) {
  if (await isUsableImageFile(outputPath)) {
    return outputPath;
  }

  const artifactPath = await findGeneratedArtifact(codexResult, startedAtMs);
  if (artifactPath) {
    await mkdir(dirname(outputPath), { recursive: true });
    await copyFile(artifactPath, outputPath);

    if (await isUsableImageFile(outputPath)) {
      return outputPath;
    }
  }

  throw new Error(
    [
      `Codex completed, but the expected image was not found at ${outputPath}.`,
      'No unambiguous generated artifact could be recovered from Codex JSON events or CODEX_HOME generated_images.',
      'Re-run with --dry-run to inspect the prompt, or inspect Codex output for generated image artifacts.',
    ].join('\n')
  );
}

/**
 * 指定 path が読み取り可能な画像ファイルとして使えるか確認します。
 *
 * ここでは拡張子と file size を検証し、空 file や想定外拡張子を成果物として扱わないようにします。
 */
async function isUsableImageFile(filePath) {
  try {
    await access(filePath, fsConstants.R_OK);
    const fileStat = await stat(filePath);
    if (
      fileStat.isFile() &&
      fileStat.size > 0 &&
      IMAGE_EXTENSIONS.has(extname(filePath).toLowerCase())
    ) {
      return true;
    }
  } catch {
    // 呼び出し元が recovery または明示 error に進めるよう false に集約します。
  }
  return false;
}

/**
 * Codex が生成した artifact を、JSON event と generated_images directory から安全に 1 件だけ特定します。
 *
 * JSON event に path が出ていればそれを優先し、取れない場合だけ実行開始時刻以降の generated_images を限定探索します。
 */
async function findGeneratedArtifact(codexResult, startedAtMs) {
  const eventCandidates = await filterExistingGeneratedImages([
    ...(codexResult.generatedImagePaths ?? []),
    ...extractGeneratedImagePathsFromText(`${codexResult.stdout}\n${codexResult.stderr}`),
  ]);

  if (eventCandidates.length === 1) {
    return eventCandidates[0];
  }

  if (eventCandidates.length > 1) {
    throw new Error(renderAmbiguousArtifactsError('Codex JSON events', eventCandidates));
  }

  const recentCandidates = await findRecentGeneratedImages(startedAtMs);
  if (recentCandidates.length === 1) {
    return recentCandidates[0];
  }

  if (recentCandidates.length > 1) {
    throw new Error(renderAmbiguousArtifactsError('generated_images directory', recentCandidates));
  }

  return null;
}

/**
 * stdout/stderr の text から generated_images 配下の画像 path を抽出します。
 *
 * JSONL を parse できる場合は構造内の全 string を調べ、parse できない text も regex で救済します。
 */
function collectGeneratedImagePathsFromText(text, output = new Set()) {
  for (const line of text.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }

    try {
      collectGeneratedImagePathsFromValue(JSON.parse(trimmed), output);
    } catch {
      // JSONL ではない行や途中 chunk は regex fallback に任せます。
    }
  }

  for (const match of text.matchAll(
    /(?:file:\/\/)?(?:~|\/[^\s"'<>]+)\/(?:\.codex\/)?generated_images\/[^\s"'<>]+?\.(?:png|jpe?g|webp)/gi
  )) {
    const normalized = normalizeGeneratedImagePath(match[0]);
    if (normalized) {
      output.add(normalized);
    }
  }

  return [...output];
}

/**
 * JSON event 内を再帰的に探索し、generated_images を指す string だけを候補へ追加します。
 */
function collectGeneratedImagePathsFromValue(value, output) {
  if (typeof value === 'string') {
    for (const candidate of extractGeneratedImagePathsFromText(value)) {
      output.add(candidate);
    }
    return;
  }

  if (Array.isArray(value)) {
    for (const item of value) {
      collectGeneratedImagePathsFromValue(item, output);
    }
    return;
  }

  if (value && typeof value === 'object') {
    for (const item of Object.values(value)) {
      collectGeneratedImagePathsFromValue(item, output);
    }
  }
}

/**
 * text から generated image path を配列として抽出します。
 */
function extractGeneratedImagePathsFromText(text) {
  return collectGeneratedImagePathsFromText(text, new Set());
}

/**
 * Codex artifact path 表記を絶対 path に正規化し、generated_images 配下だけを許可します。
 */
function normalizeGeneratedImagePath(rawPath) {
  const withoutFileScheme = rawPath.startsWith('file://')
    ? rawPath.slice('file://'.length)
    : rawPath;
  const expanded = withoutFileScheme.startsWith('~/')
    ? resolve(homedir(), withoutFileScheme.slice(2))
    : withoutFileScheme;
  const resolved = resolve(expanded);

  if (!IMAGE_EXTENSIONS.has(extname(resolved).toLowerCase())) {
    return null;
  }

  return isAllowedGeneratedImagePath(resolved) ? resolved : null;
}

/**
 * generated_images 以外の path を artifact として扱わないための境界確認です。
 */
function isAllowedGeneratedImagePath(filePath) {
  return generatedImageRoots().some((root) => {
    const pathRelativeToRoot = relative(root, filePath);
    return (
      pathRelativeToRoot &&
      !pathRelativeToRoot.startsWith('..') &&
      !pathRelativeToRoot.startsWith(sep)
    );
  });
}

/**
 * 既存候補を実ファイル・許可 root・画像拡張子で絞り、重複を除去します。
 */
async function filterExistingGeneratedImages(paths) {
  const uniquePaths = unique(paths.map(normalizeGeneratedImagePath));
  const existing = [];

  for (const imagePath of uniquePaths) {
    if (imagePath && (await isUsableImageFile(imagePath))) {
      existing.push(imagePath);
    }
  }

  return existing;
}

/**
 * Codex の generated_images directory だけを対象に、今回実行で作られた可能性が高い画像を探します。
 */
async function findRecentGeneratedImages(startedAtMs) {
  const minMtimeMs = startedAtMs - GENERATED_IMAGE_LOOKBACK_MS;
  const candidates = [];

  for (const root of generatedImageRoots()) {
    await collectRecentImages(root, minMtimeMs, candidates);
  }

  return unique(candidates).sort();
}

/**
 * generated_images root を再帰走査し、実行開始時刻以降の画像だけを候補へ追加します。
 */
async function collectRecentImages(directory, minMtimeMs, output) {
  let entries;
  try {
    entries = await readdir(directory, { withFileTypes: true });
  } catch {
    return;
  }

  for (const entry of entries) {
    const entryPath = resolve(directory, entry.name);
    if (entry.isDirectory()) {
      await collectRecentImages(entryPath, minMtimeMs, output);
      continue;
    }

    if (!entry.isFile() || !IMAGE_EXTENSIONS.has(extname(entry.name).toLowerCase())) {
      continue;
    }

    const fileStat = await stat(entryPath);
    if (
      fileStat.mtimeMs >= minMtimeMs &&
      fileStat.size > 0 &&
      isAllowedGeneratedImagePath(entryPath)
    ) {
      output.push(entryPath);
    }
  }
}

/**
 * Codex generated_images の候補 root を返します。
 *
 * `CODEX_HOME` がある環境と既定の `~/.codex` の両方を許可します。
 */
function generatedImageRoots() {
  return unique([
    process.env.CODEX_HOME ? resolve(process.env.CODEX_HOME, 'generated_images') : null,
    resolve(homedir(), '.codex', 'generated_images'),
  ]);
}

/**
 * 複数 artifact が見つかった場合は誤った画像を選ばず、候補を明示して停止します。
 */
function renderAmbiguousArtifactsError(source, candidates) {
  return [
    `Multiple generated image artifacts were found from ${source}; refusing to choose implicitly.`,
    ...candidates.map((candidate) => `- ${candidate}`),
  ].join('\n');
}

/**
 * wireframe tree を深さ優先で走査します。
 *
 * callback には node と depth を渡します。children が無い leaf や table node も要約対象にします。
 */
function walkWireframe(node, callback, depth = 0) {
  if (!node || typeof node !== 'object') {
    return;
  }

  callback(node, depth);

  if (Array.isArray(node.children)) {
    for (const child of node.children) {
      walkWireframe(child, callback, depth + 1);
    }
  }
}

/**
 * wireframe node を一行の役割説明に変換します。
 *
 * 画像生成に効く情報だけを残すため、px 値は補助情報として軽く扱い、style には変換しません。
 */
function describeNode(node) {
  const parts = [node.name ?? 'Unnamed node'];
  const role = describeRole(node);
  const layout = node.direction ? `${node.direction} layout` : null;
  const sizing = describeSizing(node);
  const children =
    Array.isArray(node.children) && node.children.length > 0
      ? `${node.children.length} child items`
      : null;

  for (const value of [role, layout, sizing, children]) {
    if (value) {
      parts.push(value);
    }
  }

  return parts.join('; ');
}

/**
 * wireframe node type を UI brief 用の役割に変換します。
 */
function describeRole(node) {
  switch (node.type) {
    case 'text':
      return node.variant ? `${node.variant} text` : 'text content';
    case 'button':
      return 'primary/action button';
    case 'link':
      return 'secondary link or navigation action';
    case 'input':
      return 'input, search, filter, or form control';
    case 'card':
      return 'grouped content card';
    case 'table':
      return 'data table';
    case 'image':
      return 'media or illustration area';
    case 'icon':
      return 'icon affordance';
    case 'divider':
      return 'section separator';
    default:
      return null;
  }
}

/**
 * sizing 情報から、主領域・固定領域・補助領域の手がかりを作ります。
 */
function describeSizing(node) {
  const values = [];
  if (node.grow) {
    values.push('flexible primary region');
  }
  if (typeof node.width === 'number') {
    values.push(`fixed width ${node.width}px`);
  }
  if (typeof node.height === 'number') {
    values.push(`height ${node.height}px`);
  }
  return values.join(', ') || null;
}

/**
 * viewport を desktop / tablet / mobile と orientation に要約します。
 */
function describeViewport(viewport) {
  const width = Number(viewport.width) || null;
  const height = Number(viewport.height) || null;
  if (!width || !height) {
    return 'unspecified viewport';
  }

  const device = width <= 640 ? 'mobile' : width <= 1024 ? 'tablet' : 'desktop';
  const orientation = width >= height ? 'landscape' : 'portrait';
  return `${device} ${orientation}, ${width}x${height}`;
}

/**
 * root layout を screen-level composition として要約します。
 */
function describeRootLayout(root) {
  const direction = root.direction ?? 'vertical';
  const childCount = Array.isArray(root.children) ? root.children.length : 0;
  return `${direction} app/screen structure with ${childCount} top-level sections`;
}

/**
 * wireframe summary を prompt section に整形します。
 */
function renderWireframeBlock(template, summary) {
  const policy = template.wireframePolicy ?? [
    'Use the wireframe as structural guidance only.',
    'Do not copy low-fidelity wireframe visual styling.',
  ];

  const lines = [
    'Wireframe reference:',
    ...policy,
    '',
    'Screen structure:',
    `- Screen: ${summary.screenName}`,
    `- Viewport: ${summary.viewport}`,
    `- Overall layout: ${summary.layout}`,
    ...summary.bullets,
  ];

  if (summary.headings.length > 0) {
    lines.push(
      '',
      'Visual/content priority cues:',
      ...summary.headings.map((heading) => `- ${heading}`)
    );
  }

  if (summary.actions.length > 0) {
    lines.push(
      '',
      'Primary and secondary actions:',
      ...summary.actions.map((action) => `- ${action}`)
    );
  }

  if (summary.tables.length > 0) {
    lines.push('', 'Data surfaces:', ...summary.tables.map((table) => `- ${table}`));
  }

  return `${lines.join('\n')}\n`;
}

/**
 * 参照画像と役割説明を prompt section に変換します。
 */
function renderImageRoles(images, imageRoles) {
  if (images.length === 0) {
    return null;
  }

  const lines = ['Input images:'];
  images.forEach((imagePath, index) => {
    const role =
      imageRoles[index] ??
      `Image ${index + 1}: reference image; use only according to the user's request.`;
    lines.push(`${role}`);
    lines.push(`Path: ${imagePath}`);
  });
  lines.push('');
  return lines.join('\n');
}

/**
 * edit template 専用の preservation 指示を生成します。
 */
function renderEditBlock(request) {
  if (!request.template.editMode) {
    return null;
  }

  return [
    'Change only:',
    request.form.changeOnly ?? "Apply the user's requested edit and avoid unrelated changes.",
    '',
    'Preserve exactly:',
    request.form.preserve ??
      'Subject identity, product geometry, label text, camera angle, framing, lighting direction, shadows, and original layout unless explicitly requested otherwise.',
    '',
    'Physical realism:',
    request.form.physicalRealism ??
      'Match scale, contact shadows, reflections, perspective, texture, and edge blending.',
    '',
  ].join('\n');
}

/**
 * dry-run 出力を作ります。
 *
 * 実行予定 command と Codex に渡す prompt を表示し、画像生成 quota を消費せずに確認できるようにします。
 */
function renderDryRun(command, prompt, request) {
  return [
    'DRY RUN: no Codex command was executed.',
    '',
    'Output path:',
    request.outputPath,
    '',
    'Command:',
    [command.bin, ...command.args].map(shellQuote).join(' '),
    '',
    'Prompt:',
    prompt,
    '',
  ].join('\n');
}

/**
 * ユーザー指定の出力 path を絶対 path に正規化します。
 *
 * 未指定の場合は `~/Pictures/codex-images/` 配下に prompt 由来の安全な file name を作ります。
 */
function resolveOutputPath(out, prompt) {
  const rawPath =
    out ||
    resolve(
      DEFAULT_OUTPUT_DIR,
      `${new Date().toISOString().replaceAll(':', '-')}-${slugify(prompt).slice(0, 56) || 'image'}.png`
    );
  const resolved = isAbsolute(rawPath) ? rawPath : resolve(rawPath);
  const extension = extname(resolved).toLowerCase();
  return IMAGE_EXTENSIONS.has(extension) ? resolved : `${resolved}.png`;
}

/**
 * required string 引数を検証します。
 */
function requiredString(value, name) {
  if (typeof value !== 'string' || value.trim().length === 0) {
    throw new Error(`--${name} is required.`);
  }
  return value.trim();
}

/**
 * 任意文字列を trim し、空文字なら未指定として扱います。
 */
function optionalString(value) {
  if (typeof value !== 'string') {
    return null;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

/**
 * 複数行または semicolon 区切りの制約欄を配列へ変換します。
 */
function splitBlock(value) {
  if (typeof value !== 'string') {
    return [];
  }
  return value
    .split(/\n|;/)
    .map((line) => line.trim())
    .filter(Boolean);
}

/**
 * timeout など正の整数値を検証します。
 */
function parsePositiveInteger(value, name) {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed <= 0) {
    throw new Error(`--${name} must be a positive integer.`);
  }
  return parsed;
}

/**
 * gpt-image-2 の代表的な size 制約に沿って入力値を検証します。
 *
 * Codex built-in tool では API パラメータ保証ではありませんが、無効な canvas 目標を prompt に入れないために検証します。
 */
function validateSize(size) {
  if (size === 'auto') {
    return;
  }

  const match = size.match(/^(\d+)x(\d+)$/);
  if (!match) {
    throw new Error(`Invalid size: ${size}. Use auto or WIDTHxHEIGHT.`);
  }

  const width = Number(match[1]);
  const height = Number(match[2]);
  const pixels = width * height;
  const longEdge = Math.max(width, height);
  const shortEdge = Math.min(width, height);

  if (width % 16 !== 0 || height % 16 !== 0) {
    throw new Error(`Invalid size: ${size}. Both edges must be multiples of 16.`);
  }
  if (longEdge > 3840) {
    throw new Error(`Invalid size: ${size}. Maximum edge must be <= 3840.`);
  }
  if (longEdge / shortEdge > 3) {
    throw new Error(`Invalid size: ${size}. Long-to-short ratio must be <= 3:1.`);
  }
  if (pixels < 655_360 || pixels > 8_294_400) {
    throw new Error(`Invalid size: ${size}. Total pixels must be between 655,360 and 8,294,400.`);
  }
}

/**
 * template ごとの標準 iteration target を返します。
 */
function defaultIterationTarget(templateName) {
  switch (templateName) {
    case 'ui-mockup':
      return 'layout hierarchy, product realism, readable typography, and implementable UI direction';
    case 'product-mockup':
    case 'product-shot':
      return 'product clarity, material realism, lighting, and label preservation';
    case 'landing-hero':
    case 'ad-creative':
      return 'message clarity, composition, negative space, and campaign fit';
    case 'infographic':
      return 'readable hierarchy, label clarity, and explanatory flow';
    default:
      return 'composition, visual clarity, and fit to the requested use case';
  }
}

/**
 * shell 表示用に引数を quote します。
 */
function shellQuote(value) {
  if (/^[A-Za-z0-9_./:=+-]+$/.test(value)) {
    return value;
  }
  return `'${value.replaceAll("'", "'\\''")}'`;
}

/**
 * option 名を kebab-case から camelCase に変換します。
 */
function camelCase(value) {
  return value.replaceAll(/-([a-z])/g, (_match, letter) => letter.toUpperCase());
}

/**
 * file name に使える slug を生成します。
 */
function slugify(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/**
 * 配列から重複を除去します。
 */
function unique(values) {
  return [...new Set(values.filter(Boolean))];
}

/**
 * 診断出力が長すぎないよう末尾だけを残します。
 */
function tail(value, maxLength = 6000) {
  return value.length > maxLength ? value.slice(value.length - maxLength) : value;
}

/**
 * stdout/stderr buffer が肥大化しすぎないよう制限します。
 */
function limitBuffer(value, maxLength = 64_000) {
  return value.length > maxLength ? value.slice(value.length - maxLength) : value;
}

/**
 * help text を返します。
 */
function helpText() {
  return `generate-image ${VERSION}

Usage:
  node .opencode/skills/generate-image/scripts/generate-image.mjs --prompt <first-line directive> [form options]

Options:
  --template <name>            Template name. Default: general
  --prompt <text>              First-line directive: deliverable type + visual mode + use case. Required
  --purpose <text>             Purpose section
  --canvas <text>              Canvas section: ratio, medium, crop, safe area, whitespace
  --subject <text>             Subject section
  --composition <text>         Composition section
  --style <text>               Style section
  --text <text>                Text section for exact in-image text rules
  --typography <text>          Typography section
  --details <text>             Details section
  --preserve <text>            Preserve section
  --change-only <text>         Edit-only Change only section
  --physical-realism <text>    Edit-only Physical realism section
  --constraint <text>          Add one constraint. Repeatable
  --constraints <text>         Add newline- or semicolon-separated constraints
  --out <path>                 Output image path. Default: ~/Pictures/codex-images/<timestamp>-<slug>.png
  --quality <value>            low | medium | high | auto. Default: medium
  --size <value>               auto or WIDTHxHEIGHT. Default: auto
  --wireframe <path>           Optional .wireframe.json or .wireframe.html reference, mainly for ui-mockup
  --image <path>               Optional reference/edit image. Repeatable
  --image-role <text>          Role label for each --image. Repeatable
  --iteration-target <text>    What this generation should optimize
  --timeout-ms <number>        Codex timeout. Default: 600000
  --dry-run                    Print command and prompt without running Codex
  --help                       Show this help
  --version                    Show version

Templates:
  ${Object.keys(TEMPLATES).join('\n  ')}
`;
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});

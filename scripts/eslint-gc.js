#!/usr/bin/env node

import { ESLint } from 'eslint';

const parseArgs = (args) => {
  const options = {
    fix: false,
    allowInlineConfig: true,
    maxWarnings: -1,
    patterns: [],
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args.at(i);
    if (arg === '--fix') {
      options.fix = true;
    } else if (arg === '--no-inline-config') {
      options.allowInlineConfig = false;
    } else if (arg === '--max-warnings') {
      i++;
      const next = args.at(i);
      if (next !== undefined) {
        options.maxWarnings = Number(next);
      }
    } else if (!arg.startsWith('-')) {
      options.patterns.push(arg);
    }
  }

  return options;
};

const main = async () => {
  const args = parseArgs(process.argv.slice(2));

  const eslint = new ESLint({
    fix: args.fix,
    allowInlineConfig: args.allowInlineConfig,
  });

  const results = await eslint.lintFiles(args.patterns);

  if (args.fix) {
    await ESLint.outputFixes(results);
  }

  const formatter = await eslint.loadFormatter();
  const resultText = await formatter.format(results);
  if (resultText) {
    console.log(resultText);
  }

  const errorCount = results.reduce((sum, r) => sum + r.errorCount + r.fatalErrorCount, 0);
  const warningCount = results.reduce((sum, r) => sum + r.warningCount, 0);

  let exitCode = 0;
  if (errorCount > 0) {
    exitCode = 1;
  } else if (args.maxWarnings >= 0 && warningCount > args.maxWarnings) {
    exitCode = 1;
  }

  if (typeof global.gc === 'function') {
    global.gc();
  }

  process.exit(exitCode);
};

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

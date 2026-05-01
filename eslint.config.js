import { FlatCompat } from '@eslint/eslintrc';
import js from '@eslint/js';
import boundaries from 'eslint-plugin-boundaries';
import deprecation from 'eslint-plugin-deprecation';
import eslintComments from 'eslint-plugin-eslint-comments';
import importPlugin from 'eslint-plugin-import';
import security from 'eslint-plugin-security';
import sonarjs from 'eslint-plugin-sonarjs';
import svelte from 'eslint-plugin-svelte';
import unicorn from 'eslint-plugin-unicorn';
import tseslint from 'typescript-eslint';

import maxlinesConfig from './.eslintrc-maxlines.json' with { type: 'json' };
import uiSvelteConfig from './packages/frontend/ui/svelte.config.js';

const compat = new FlatCompat();

const frontendAppSourceFiles = [
  'packages/frontend/app/src/**/*.{ts,js}',
  'packages/frontend/app/src/**/*.svelte',
  'packages/frontend/app/src/**/*.svelte.ts',
  'packages/frontend/app/src/**/*.svelte.js',
];

const frontendWebSourceFiles = [
  'packages/web/src/**/*.{ts,js}',
  'packages/web/src/**/*.svelte',
  'packages/web/src/**/*.svelte.ts',
  'packages/web/src/**/*.svelte.js',
];

const frontendDomainSourceFiles = [
  'packages/frontend/domain/src/**/*.ts',
  'packages/frontend/domain/src/**/*.svelte.ts',
  'packages/frontend/domain/src/**/*.svelte.js',
];

const frontendUiSourceFiles = [
  'packages/frontend/ui/src/**/*.ts',
  'packages/frontend/ui/src/**/*.svelte',
  'packages/frontend/ui/src/**/*.svelte.ts',
  'packages/frontend/ui/src/**/*.svelte.js',
];

const frontendSvelteFiles = [
  'packages/frontend/**/*.svelte',
  'packages/frontend/**/*.svelte.ts',
  'packages/frontend/**/*.svelte.js',
  'packages/web/**/*.svelte',
  'packages/web/**/*.svelte.ts',
  'packages/web/**/*.svelte.js',
];

const frontendAppRoutePageFiles = ['packages/frontend/app/src/routes/**/*.svelte'];

const frontendWebRoutePageFiles = ['packages/web/src/routes/**/*.svelte'];

const frontendRoutePageFiles = [...frontendAppRoutePageFiles, ...frontendWebRoutePageFiles];

const frontendAppComponentFiles = [
  'packages/frontend/app/src/components/**/*.{ts,js}',
  'packages/frontend/app/src/components/**/*.svelte',
  'packages/frontend/app/src/lib/**/*.{ts,js}',
  'packages/frontend/app/src/lib/**/*.svelte',
];

const frontendWebComponentFiles = [
  'packages/web/src/components/**/*.{ts,js}',
  'packages/web/src/components/**/*.svelte',
  'packages/web/src/lib/**/*.{ts,js}',
  'packages/web/src/lib/**/*.svelte',
];

const frontendComponentFiles = [...frontendAppComponentFiles, ...frontendWebComponentFiles];

const frontendDomainHookFiles = [];

const frontendDomainHookSvelteFiles = [
  'packages/frontend/domain/src/**/*.svelte.ts',
  'packages/frontend/domain/src/**/*.svelte.js',
];

const frontendDomainPlainTsFiles = ['packages/frontend/domain/src/**/*.ts'];

const frontendNonReactSourceFiles = [
  ...frontendAppSourceFiles,
  ...frontendWebSourceFiles,
  ...frontendDomainSourceFiles,
  ...frontendUiSourceFiles,
];

const frontendWebSvelteKitImportFiles = [
  ...frontendWebSourceFiles,
  'packages/web/src/**/*.svelte.ts',
  'packages/web/src/**/*.svelte.js',
];

const frontendAppSvelteKitImportFiles = [
  ...frontendAppSourceFiles,
  'packages/frontend/app/src/**/*.svelte.ts',
  'packages/frontend/app/src/**/*.svelte.js',
];

const frontendWebSvelteKitRouteModuleFiles = [
  'packages/web/src/routes/**/+page.{ts,js}',
  'packages/web/src/routes/**/+layout.{ts,js}',
];

const frontendAppSvelteKitRouteModuleFiles = [
  'packages/frontend/app/src/routes/**/+page.{ts,js}',
  'packages/frontend/app/src/routes/**/+layout.{ts,js}',
];

const frontendWebSvelteKitHookModuleFiles = [
  'packages/web/src/hooks.{ts,js}',
  'packages/web/src/hooks.client.{ts,js}',
  'packages/web/src/hooks.server.{ts,js}',
];

const frontendAppSvelteKitHookModuleFiles = [
  'packages/frontend/app/src/hooks.{ts,js}',
  'packages/frontend/app/src/hooks.client.{ts,js}',
  'packages/frontend/app/src/hooks.server.{ts,js}',
];

const frontendWebSvelteKitPageServerModuleFiles = [
  'packages/web/src/routes/**/+page.server.{ts,js}',
];

const frontendAppSvelteKitPageServerModuleFiles = [
  'packages/frontend/app/src/routes/**/+page.server.{ts,js}',
];

const frontendAppSvelteKitServerOnlyFiles = [
  'packages/frontend/app/src/routes/**/+server.{ts,js}',
  'packages/frontend/app/src/routes/**/+page.server.{ts,js}',
  'packages/frontend/app/src/routes/**/+layout.server.{ts,js}',
  'packages/frontend/app/src/hooks.server.{ts,js}',
  'packages/frontend/app/src/lib/server/**/*.{ts,js,svelte}',
  'packages/frontend/app/src/lib/server/**/*.svelte.{ts,js}',
];

const exportTsdocPlugin = {
  rules: {
    'require-export-tsdoc': {
      meta: {
        type: 'problem',
        docs: {
          description: 'Require TSDoc comments for exported declarations.',
        },
        schema: [],
        messages: {
          missing:
            'エクスポートする{{target}}には直前に TSDoc コメント (/** ... */) を付けてください。',
        },
      },
      create(context) {
        const sourceCode = context.getSourceCode();

        const isTsdocCommentBefore = (node) => {
          const comments = sourceCode.getCommentsBefore(node);
          if (comments.length === 0) {
            return false;
          }
          const last = comments[comments.length - 1];
          const isAdjacent = node.loc.start.line - last.loc.end.line <= 1;
          const isTsdoc = last.type === 'Block' && last.value.startsWith('*');
          return isAdjacent && isTsdoc;
        };

        const hasTsdocComment = (node) => {
          if (isTsdocCommentBefore(node)) {
            return true;
          }
          const parent = node.parent;
          if (
            parent &&
            (parent.type === 'ExportNamedDeclaration' || parent.type === 'ExportDefaultDeclaration')
          ) {
            return isTsdocCommentBefore(parent);
          }
          return false;
        };

        const reportIfMissing = (targetNode, label) => {
          if (hasTsdocComment(targetNode)) return;
          context.report({ node: targetNode, messageId: 'missing', data: { target: label } });
        };

        const getExportInfo = (node) => {
          const decl = node.declaration;
          if (!decl) return null;
          switch (decl.type) {
            case 'FunctionDeclaration':
              return { target: decl, label: '関数' };
            case 'ClassDeclaration':
              return { target: decl, label: 'クラス' };
            case 'TSEnumDeclaration':
              return { target: decl, label: 'enum' };
            case 'TSInterfaceDeclaration':
              return { target: decl, label: 'インターフェース' };
            case 'TSTypeAliasDeclaration':
              return { target: decl, label: '型' };
            case 'VariableDeclaration':
              return { target: decl, label: '変数/定数' };
            default:
              return { target: decl, label: '値' };
          }
        };

        const getDefaultExportInfo = (node) => {
          const decl = node.declaration;
          if (!decl) return null;
          const target = decl.type === 'Identifier' ? node : decl;
          return { target, label: 'default export' };
        };

        return {
          ExportNamedDeclaration(node) {
            const info = getExportInfo(node);
            if (!info) return;
            reportIfMissing(info.target, info.label);
          },
          ExportDefaultDeclaration(node) {
            const info = getDefaultExportInfo(node);
            if (!info) return;
            reportIfMissing(info.target, info.label);
          },
        };
      },
    },
  },
};

const unwrapStaticExpression = (node) => {
  let current = node;

  while (
    current &&
    (current.type === 'TSAsExpression' ||
      current.type === 'TSSatisfiesExpression' ||
      current.type === 'TSNonNullExpression' ||
      current.type === 'ParenthesizedExpression')
  ) {
    current = current.expression;
  }

  return current;
};

const collectTopLevelBindings = (program) => {
  const bindings = new Map();

  const addBinding = (name, binding) => {
    if (name === '' || bindings.has(name)) {
      return;
    }

    bindings.set(name, binding);
  };

  for (const statement of program.body) {
    const declaration =
      statement.type === 'ExportNamedDeclaration' ? statement.declaration : statement;

    if (!declaration) {
      continue;
    }

    if (declaration.type === 'VariableDeclaration') {
      for (const declarator of declaration.declarations) {
        if (declarator.id.type === 'Identifier') {
          addBinding(declarator.id.name, {
            localName: declarator.id.name,
            node: declarator,
            init: declarator.init,
            kind: 'variable',
          });
        }
      }

      continue;
    }

    if ('id' in declaration && declaration.id?.type === 'Identifier') {
      addBinding(declaration.id.name, {
        localName: declaration.id.name,
        node: declaration,
        init: null,
        kind: declaration.type,
      });
    }
  }

  return bindings;
};

const getExportedBindings = (program) => {
  const bindings = [];
  const topLevelBindings = collectTopLevelBindings(program);

  for (const statement of program.body) {
    if (statement.type !== 'ExportNamedDeclaration') {
      continue;
    }

    if (statement.declaration) {
      const declaration = statement.declaration;

      if (declaration.type === 'VariableDeclaration') {
        for (const declarator of declaration.declarations) {
          if (declarator.id.type === 'Identifier') {
            bindings.push({
              exportedName: declarator.id.name,
              localName: declarator.id.name,
              node: declarator,
              init: declarator.init,
              kind: 'variable',
            });
          }
        }

        continue;
      }

      if ('id' in declaration && declaration.id?.type === 'Identifier') {
        bindings.push({
          exportedName: declaration.id.name,
          localName: declaration.id.name,
          node: declaration,
          init: null,
          kind: declaration.type,
        });
      }

      continue;
    }

    for (const specifier of statement.specifiers) {
      if (specifier.exported.type === 'Identifier') {
        const localName = specifier.local.type === 'Identifier' ? specifier.local.name : null;
        const localBinding = localName ? topLevelBindings.get(localName) : null;

        bindings.push({
          exportedName: specifier.exported.name,
          localName,
          node: specifier,
          init: localBinding?.init ?? null,
          kind: localBinding?.kind ?? 'specifier',
        });
      }
    }
  }

  return bindings;
};

const isForbiddenSvelteKitImportSource = (source) => {
  if (
    source === '$app/server' ||
    source.startsWith('$app/server/') ||
    source === '$env/static/private' ||
    source.startsWith('$env/static/private/') ||
    source === '$env/dynamic/private' ||
    source.startsWith('$env/dynamic/private/') ||
    source === '$lib/server' ||
    source.startsWith('$lib/server/')
  ) {
    return true;
  }

  if (
    (source.startsWith('./') || source.startsWith('../')) &&
    /(^|\/)lib\/server(?:\/|$)/.test(source)
  ) {
    return true;
  }

  return /(^|\/)[^/]+\.server(?:\.[^/]+)?$/.test(source);
};

const sveltekitAppPolicyPlugin = {
  rules: {
    'no-forbidden-imports': {
      meta: {
        type: 'problem',
        schema: [],
        messages: {
          forbiddenSource:
            'frontend app では SvelteKit の server 専用 module `{{source}}` を import しないでください。公開 SSR は route module、認証 `/app/*` は CSR で構成してください。',
          forbiddenKitImport:
            'frontend app では SvelteKit の server API `{{name}}` を import しないでください。',
        },
      },
      create(context) {
        return {
          ImportDeclaration(node) {
            if (typeof node.source.value !== 'string') {
              return;
            }

            const source = node.source.value;

            if (isForbiddenSvelteKitImportSource(source)) {
              context.report({ node: node.source, messageId: 'forbiddenSource', data: { source } });
            }

            if (source !== '@sveltejs/kit') {
              return;
            }

            const forbiddenImports = new Set([
              'Actions',
              'Handle',
              'HandleFetch',
              'LayoutServerLoad',
              'PageServerLoad',
              'RequestHandler',
            ]);

            for (const specifier of node.specifiers) {
              if (
                specifier.type === 'ImportSpecifier' &&
                forbiddenImports.has(specifier.imported.name)
              ) {
                context.report({
                  node: specifier,
                  messageId: 'forbiddenKitImport',
                  data: { name: specifier.imported.name },
                });
              }
            }
          },
        };
      },
    },
    'no-export-names': {
      meta: {
        type: 'problem',
        schema: [
          {
            type: 'object',
            properties: {
              message: { type: 'string' },
              names: {
                type: 'array',
                items: { type: 'string' },
              },
            },
            additionalProperties: false,
          },
        ],
        messages: {
          forbiddenExport: 'frontend app policy により export `{{name}}` を禁止します。',
        },
      },
      create(context) {
        const option = context.options[0] ?? {};
        const forbiddenNames = new Set(option.names ?? []);
        const customMessage = option.message;

        return {
          Program(node) {
            for (const binding of getExportedBindings(node)) {
              if (!forbiddenNames.has(binding.exportedName)) {
                continue;
              }

              context.report({
                node: binding.node,
                ...(customMessage
                  ? { message: customMessage, data: { name: binding.exportedName } }
                  : { messageId: 'forbiddenExport', data: { name: binding.exportedName } }),
              });
            }
          },
        };
      },
    },
    'require-auth-layout-mode': {
      meta: {
        type: 'problem',
        schema: [],
        messages: {
          missingSsr:
            '認証 route の親 layout では `export const ssr = false` を必須にしてください。`/app/*` は SSR しません。',
          missingCsr:
            '認証 route の親 layout では `export const csr = true` を必須にしてください。`/app/*` は CSR 前提です。',
          invalidSsr:
            '認証 route の親 layout の `ssr` は `false` の boolean literal だけを許可します。',
          invalidCsr:
            '認証 route の親 layout の `csr` は `true` の boolean literal だけを許可します。',
        },
      },
      create(context) {
        return {
          Program(node) {
            const bindings = getExportedBindings(node);
            const exportedMap = new Map(bindings.map((binding) => [binding.exportedName, binding]));

            const ssrBinding = exportedMap.get('ssr');
            if (!ssrBinding) {
              context.report({ node, messageId: 'missingSsr' });
            } else {
              const ssrInit = unwrapStaticExpression(ssrBinding.init);
              if (
                ssrBinding.kind !== 'variable' ||
                ssrInit?.type !== 'Literal' ||
                ssrInit.value !== false
              ) {
                context.report({ node: ssrBinding.node, messageId: 'invalidSsr' });
              }
            }

            const csrBinding = exportedMap.get('csr');
            if (!csrBinding) {
              context.report({ node, messageId: 'missingCsr' });
            } else {
              const csrInit = unwrapStaticExpression(csrBinding.init);
              if (
                csrBinding.kind !== 'variable' ||
                csrInit?.type !== 'Literal' ||
                csrInit.value !== true
              ) {
                context.report({ node: csrBinding.node, messageId: 'invalidCsr' });
              }
            }
          },
        };
      },
    },
  },
};

const frontendSvelte5Plugin = {
  rules: {
    'no-legacy-syntax': {
      meta: {
        type: 'problem',
        schema: [],
      },
      create(context) {
        const sourceCode = context.getSourceCode();
        const checks = [
          {
            pattern: /\bon:[A-Za-z][\w-]*\s*=/g,
            message:
              'Svelte 5 では `on:` ディレクティブを使わず、`onclick` などの property 形式を使ってください。',
          },
          {
            pattern: /<slot\b/g,
            message:
              'Svelte 5 では `<slot>` を使わず、snippet と `{@render ...}` を使ってください。',
          },
          {
            pattern: /\$\$slots\b/g,
            message:
              'Svelte 5 では `$$slots` を使わず、snippet と `{@render ...}` を使ってください。',
          },
          {
            pattern: /\$\$restProps\b/g,
            message: 'Svelte 5 では `$$restProps` を使わず `$props()` を使ってください。',
          },
          {
            pattern: /\bexport let\b/g,
            message: 'Svelte 5 では `export let` を使わず `$props()` を使ってください。',
          },
          {
            pattern: /(^|\n)\s*\$:\s/g,
            message: 'Svelte 5 では `$:` を使わず `$derived` または `$effect` を使ってください。',
          },
          {
            pattern: /\bcreateEventDispatcher\b/g,
            message:
              'Svelte 5 では `createEventDispatcher` ではなく callback props を使ってください。',
          },
        ];

        return {
          'Program:exit'(node) {
            for (const check of checks) {
              for (const match of sourceCode.text.matchAll(check.pattern)) {
                const start = sourceCode.getLocFromIndex(match.index ?? 0);
                const end = sourceCode.getLocFromIndex((match.index ?? 0) + match[0].length);

                context.report({
                  node,
                  loc: { start, end },
                  message: check.message,
                });
              }
            }
          },
        };
      },
    },
  },
};

const frontendAppPrimitiveUiPlugin = {
  rules: {
    'no-primitive-tags': {
      meta: {
        type: 'problem',
        schema: [],
        messages: {
          forbidden:
            'packages/frontend/app では `<{{tag}}>` を直書きしないでください。まず `@www-template/ui/components` の既存 component を使い、足りなければ `packages/frontend/ui` を発展させてから app で compose してください。',
        },
      },
      create(context) {
        const ignoredRangePatterns = [
          /<script\b[\S\s]*?<\/script>/gi,
          /<style\b[\S\s]*?<\/style>/gi,
          /<!--[\S\s]*?-->/g,
        ];

        const forbiddenTagPattern = /<(?!!--)\s*(button|input|select|textarea|table)\b/g;

        const collectIgnoredRanges = (sourceText) => {
          const ranges = [];

          for (const pattern of ignoredRangePatterns) {
            for (const match of sourceText.matchAll(pattern)) {
              if (match.index === undefined) {
                continue;
              }

              ranges.push({
                start: match.index,
                end: match.index + match[0].length,
              });
            }
          }

          return ranges;
        };

        const isIgnoredIndex = (index, ranges) =>
          ranges.some((range) => index >= range.start && index < range.end);

        return {
          'Program:exit'(node) {
            const sourceText = context.getSourceCode().text;
            const ignoredRanges = collectIgnoredRanges(sourceText);

            for (const match of sourceText.matchAll(forbiddenTagPattern)) {
              if (match.index === undefined || isIgnoredIndex(match.index, ignoredRanges)) {
                continue;
              }

              const tag = match[1]?.toLowerCase();

              if (!tag) {
                continue;
              }

              const start = context.getSourceCode().getLocFromIndex(match.index);
              const end = context.getSourceCode().getLocFromIndex(match.index + match[0].length);

              context.report({
                node,
                loc: { start, end },
                messageId: 'forbidden',
                data: { tag },
              });
            }
          },
        };
      },
    },
  },
};

const frontendCssPolicyPlugin = {
  rules: {
    'no-svelte-style-tag': {
      meta: {
        type: 'problem',
        docs: { description: 'Disallow <style> tags in Svelte files' },
        schema: [],
        messages: {
          styleTag:
            '<style> tags are forbidden in Svelte files. Use Tailwind utilities or @layer components in CSS files instead.',
        },
      },
      create(context) {
        return {
          SvelteStyleElement(node) {
            context.report({ node, messageId: 'styleTag' });
          },
        };
      },
    },
    'no-tailwind-arbitrary-values': {
      meta: {
        type: 'problem',
        docs: { description: 'Disallow Tailwind CSS arbitrary value syntax' },
        schema: [],
        messages: {
          arbitraryValue:
            'Tailwind arbitrary value "{{value}}" is forbidden. Use Design Tokens or @layer components instead.',
        },
      },
      create(context) {
        const arbitraryValueRegex = /\b\w+-\[[^\]]+]/g;

        function reportMatches(node, text) {
          if (typeof text !== 'string') return;
          for (const match of text.matchAll(arbitraryValueRegex)) {
            context.report({
              node,
              messageId: 'arbitraryValue',
              data: { value: match[0] },
            });
          }
        }

        return {
          Literal(node) {
            const parent = node.parent;
            if (
              parent?.type === 'JSXAttribute' &&
              (parent.name?.name === 'class' || parent.name?.name === 'className')
            ) {
              reportMatches(node, node.value);
            }
            if (parent?.type === 'SvelteAttribute' && parent.key?.name === 'class') {
              reportMatches(node, node.value);
            }
            if (parent?.type === 'SvelteDirective' && parent.name?.name === 'class') {
              reportMatches(node, node.value);
            }
          },
          SvelteAttribute(node) {
            if (node.key?.name !== 'class') return;
            for (const v of node.value) {
              if (v.type === 'Literal') {
                reportMatches(v, v.value);
              }
              if (
                v.type === 'SvelteExpressionContainer' &&
                v.expression?.type === 'TemplateLiteral'
              ) {
                for (const quasi of v.expression.quasis) {
                  reportMatches(quasi, quasi.value.raw);
                }
              }
              if (v.type === 'SvelteExpressionContainer' && v.expression?.type === 'Literal') {
                reportMatches(v.expression, v.expression.value);
              }
            }
          },
          SvelteDirective(node) {
            if (node.name?.name !== 'class') return;
            for (const v of node.value) {
              if (v.type === 'Literal') {
                reportMatches(v, v.value);
              }
            }
          },
        };
      },
    },
  },
};

const isImportMetaEnvChain = (node) => {
  if (!node || node.type !== 'MemberExpression' || node.computed) {
    return false;
  }

  if (
    node.object.type === 'MetaProperty' &&
    node.object.meta.name === 'import' &&
    node.object.property.name === 'meta' &&
    node.property.type === 'Identifier' &&
    node.property.name === 'env'
  ) {
    return true;
  }

  return node.object.type === 'MemberExpression' && isImportMetaEnvChain(node.object);
};

const frontendDomainPurityPlugin = {
  rules: {
    'no-runtime-env': {
      meta: {
        type: 'problem',
        schema: [],
        messages: {
          forbidden:
            'frontend domain では `import.meta.env` に直接依存しないでください。runtime 条件分岐は app 層か adapter に寄せてください。',
        },
      },
      create(context) {
        return {
          MemberExpression(node) {
            if (!isImportMetaEnvChain(node)) {
              return;
            }

            if (
              node.parent?.type === 'MemberExpression' &&
              node.parent.object === node &&
              isImportMetaEnvChain(node.parent)
            ) {
              return;
            }

            context.report({ node, messageId: 'forbidden' });
          },
        };
      },
    },
  },
};

export default tseslint.config(
  // 除外対象
  {
    ignores: [
      '**/.svelte-kit/**',
      '**/coverage/**',
      '**/playwright-report/**',
      '**/test-results/**',
      'scripts/eslint-gc.js',
    ],
  },

  // ベース設定
  js.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  ...compat.extends('plugin:import/typescript'),
  ...svelte.configs.recommended,

  // 全体設定
  {
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
  },

  {
    files: frontendSvelteFiles,
    plugins: {
      'frontend-svelte5': frontendSvelte5Plugin,
    },
    languageOptions: {
      globals: {
        document: 'readonly',
        localStorage: 'readonly',
        navigator: 'readonly',
        sessionStorage: 'readonly',
        window: 'readonly',
      },
      parserOptions: {
        projectService: true,
        extraFileExtensions: ['.svelte'],
        parser: tseslint.parser,
        svelteConfig: uiSvelteConfig,
      },
    },
    rules: {
      'frontend-svelte5/no-legacy-syntax': 'error',
      'svelte/valid-compile': 'error',
      'svelte/require-each-key': 'error',
      'svelte/no-target-blank': 'error',
      'svelte/no-navigation-without-resolve': 'off',
      'svelte/no-at-html-tags': 'error',
      'svelte/prefer-writable-derived': 'off',
    },
  },

  {
    files: [
      'packages/frontend/app/src/routes/**/*.svelte',
      'packages/frontend/app/src/components/**/*.svelte',
      'packages/frontend/app/src/lib/**/*.svelte',
    ],
    plugins: {
      'frontend-app-primitive-ui': frontendAppPrimitiveUiPlugin,
    },
    rules: {
      'frontend-app-primitive-ui/no-primitive-tags': 'error',
    },
  },

  {
    files: ['packages/frontend/ui/src/SafeHTML.svelte'],
    rules: {
      'svelte/no-at-html-tags': 'off',
    },
  },

  // グローバルルール設定
  {
    plugins: {
      import: importPlugin,
      unicorn: unicorn,
      'eslint-comments': eslintComments,
      boundaries: boundaries,
      deprecation: deprecation,
      security: security,
      sonarjs: sonarjs,
    },
    settings: {
      'import/resolver': {
        typescript: {
          alwaysTryTypes: true,
          noWarnOnMultipleProjects: true,
          project: ['./tsconfig.base.json', './packages/*/*/tsconfig.json'],
        },
      },
      'boundaries/elements': [
        {
          type: 'typespec-openapi',
          pattern: 'packages/typespec/openapi/openapi.json',
          mode: 'full',
        },
        { type: 'frontend-api', pattern: 'packages/frontend/api/src/**/*', mode: 'full' },
        { type: 'frontend-domain', pattern: 'packages/frontend/domain/src/**/*', mode: 'full' },
        { type: 'frontend-app', pattern: 'packages/frontend/app/src/**/*', mode: 'full' },
        { type: 'frontend-web', pattern: 'packages/web/src/**/*', mode: 'full' },
        { type: 'ui', pattern: 'packages/frontend/ui/src/**/*', mode: 'full' },
        {
          type: 'domain-auth',
          pattern: 'packages/frontend/domain/src/auth/**/*',
          mode: 'full',
        },
        {
          type: 'domain-status',
          pattern: 'packages/frontend/domain/src/status/**/*',
          mode: 'full',
        },
        {
          type: 'domain-observability',
          pattern: 'packages/frontend/domain/src/observability/**/*',
          mode: 'full',
        },
      ],
    },
    rules: {
      // ===== TypeScript 厳格化 =====
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unsafe-assignment': 'error',
      '@typescript-eslint/no-unsafe-call': 'error',
      '@typescript-eslint/no-unsafe-member-access': 'error',
      '@typescript-eslint/no-unsafe-return': 'error',
      '@typescript-eslint/no-unsafe-argument': 'error',
      '@typescript-eslint/strict-boolean-expressions': [
        'error',
        {
          allowString: false,
          allowNumber: false,
          allowNullableObject: false,
        },
      ],
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/await-thenable': 'error',
      '@typescript-eslint/no-misused-promises': 'error',
      '@typescript-eslint/prefer-nullish-coalescing': 'error',
      '@typescript-eslint/prefer-optional-chain': 'error',
      '@typescript-eslint/no-unnecessary-condition': 'error',
      '@typescript-eslint/no-confusing-void-expression': 'error',
      '@typescript-eslint/no-unnecessary-type-assertion': 'error',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          prefer: 'type-imports',
          fixStyle: 'inline-type-imports',
        },
      ],

      // ===== ESLint 無効化コメントの制限 =====
      'eslint-comments/no-unused-disable': 'error',
      'eslint-comments/disable-enable-pair': 'error',
      'eslint-comments/require-description': [
        'error',
        {
          ignore: [],
        },
      ],
      'eslint-comments/no-use': 'error',

      // ===== Import/Export =====
      'import/no-duplicates': 'error',
      'import/no-unresolved': 'off', // TypeScript が解決するのでオフ
      'import/extensions': [
        'error',
        'ignorePackages',
        {
          ts: 'never',
          tsx: 'never',
          js: 'never',
          jsx: 'never',
        },
      ],
      'import/order': [
        'error',
        {
          groups: [
            'builtin',
            'external',
            'internal',
            'parent',
            'sibling',
            'index',
            'object',
            'type',
          ],
          'newlines-between': 'always',
          alphabetize: {
            order: 'asc',
            caseInsensitive: true,
          },
          pathGroups: [
            {
              pattern: '@www-template/**',
              group: 'internal',
              position: 'after',
            },
            {
              pattern: '@www-template/ui/**',
              group: 'internal',
              position: 'after',
            },
            { pattern: '@ui/**', group: 'internal', position: 'after' },
          ],
          pathGroupsExcludedImportTypes: ['builtin'],
        },
      ],

      // ===== Unicorn (厳選) =====
      'unicorn/better-regex': 'error',
      'unicorn/catch-error-name': 'error',
      'unicorn/no-array-for-each': 'error',
      'unicorn/prefer-node-protocol': 'error',
      'unicorn/prefer-type-error': 'error',
      'unicorn/throw-new-error': 'error',

      // ===== Clean Architecture boundaries =====
      'boundaries/element-types': [
        'error',
        {
          default: 'disallow',
          message: 'Clean Architecture violation: %{from} is not allowed to import from %{target}.',
          rules: [
            {
              from: ['frontend-api'],
              allow: ['frontend-api'],
            },
            {
              from: ['frontend-domain'],
              allow: ['frontend-domain', 'frontend-api'],
            },
            {
              from: ['frontend-app'],
              allow: ['frontend-app', 'frontend-domain', 'ui'],
            },
            {
              from: ['frontend-web'],
              allow: ['frontend-web', 'ui'],
            },
            {
              from: ['ui'],
              allow: ['ui'],
            },
          ],
        },
      ],

      // Domain / UseCase は外部ライブラリに依存しない
      'boundaries/external': [
        'error',
        {
          default: 'allow',
          rules: [],
        },
      ],

      // ===== セキュリティ =====
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-new-func': 'error',
      'no-script-url': 'error',
      'security/detect-object-injection': 'warn',
      'security/detect-non-literal-regexp': 'warn',
      'security/detect-possible-timing-attacks': 'warn',

      // ===== Sonar (コードスメル) =====
      'sonarjs/no-identical-functions': 'warn',
      'sonarjs/no-duplicate-string': [
        'warn',
        {
          threshold: 5,
        },
      ],
      'sonarjs/cognitive-complexity': ['warn', 30],

      // ===== コード品質 =====
      'no-console': 'warn',
      'no-debugger': 'error',
      'no-alert': 'error',
      'no-var': 'error',
      'prefer-const': 'error',
      'prefer-arrow-callback': 'error',
      'no-unused-vars': 'off', // TypeScript が処理
      // ファイル/関数の肥大化防止
    },
  },

  {
    files: frontendSvelteFiles,
    ...tseslint.configs.disableTypeChecked,
    rules: {
      ...tseslint.configs.disableTypeChecked.rules,
      'import/order': 'off',
      'import/extensions': 'off',
      'prefer-const': 'off',
      'no-undef': 'off',
      'security/detect-object-injection': 'off',
      '@typescript-eslint/consistent-type-definitions': 'off',
      '@typescript-eslint/no-base-to-string': 'off',
      '@typescript-eslint/no-useless-default-assignment': 'off',
      '@typescript-eslint/restrict-template-expressions': 'off',
      '@typescript-eslint/strict-boolean-expressions': 'off',
    },
  },

  // Boundaries: 層定義外のファイルや依存を禁止
  {
    files: [
      ...frontendAppSourceFiles,
      ...frontendWebSourceFiles,
      ...frontendDomainSourceFiles,
      ...frontendUiSourceFiles,
    ],
    ignores: [
      // .svelte ファイルは SvelteKit 仮想モジュール ($app/*, $lib/*) を import するため除外
      'packages/frontend/**/*.svelte',
      'packages/frontend/**/*.svelte.ts',
      'packages/frontend/**/*.svelte.js',
      'packages/web/**/*.svelte',
      'packages/web/**/*.svelte.ts',
      'packages/web/**/*.svelte.js',
    ],
    rules: {
      'boundaries/no-unknown-files': 'error',
      'boundaries/no-unknown': 'error',
      'boundaries/no-ignored': 'error',
    },
  },

  // エクスポートは TSDoc 必須（再エクスポートは対象外）
  {
    files: ['packages/**/src/**/*.{ts,tsx}'],
    ignores: [
      'packages/frontend/api/src/generated/**/*.{ts,tsx}',
      '**/*.test.ts',
      '**/*.test.tsx',
      '**/*.stories.ts',
      '**/*.stories.tsx',
      '**/*.spec.ts',
      '**/*.spec.tsx',
    ],
    plugins: {
      'export-tsdoc': exportTsdocPlugin,
    },
    rules: {
      'export-tsdoc/require-export-tsdoc': 'error',
    },
  },

  // packages 配下は import で拡張子 .js を禁止
  {
    files: ['packages/**/*.ts', 'packages/**/*.tsx'],
    ignores: [
      'packages/frontend/api/src/generated/**/*.ts',
      'packages/frontend/api/src/generated/**/*.tsx',
      'packages/**/*.svelte.ts',
      'packages/**/*.svelte.js',
    ],
    rules: {
      'import/extensions': [
        'error',
        'ignorePackages',
        {
          ts: 'never',
          tsx: 'never',
          js: 'never',
          jsx: 'never',
          mjs: 'never',
          cjs: 'never',
        },
      ],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['*.js', '*.mjs', '*.cjs', '**/*.js', '**/*.mjs', '**/*.cjs'],
              message: 'import パスに .js / .mjs / .cjs 拡張子を付けないでください。',
            },
          ],
        },
      ],
    },
  },
  // TS/TSX の長さ制約（別ファイルの JSON から読み込み）
  maxlinesConfig,

  // テストファイルは長さ制約を除外
  {
    files: [
      '**/*.test.{ts,tsx}',
      '**/*.spec.{ts,tsx}',
      '**/*.test.svelte',
      '**/*.test.svelte.ts',
      '**/*.spec.svelte',
      '**/*.spec.svelte.ts',
    ],
    rules: {
      'max-lines': 'off',
      'max-lines-per-function': 'off',
    },
  },

  // Presentation 層から API パッケージを直接参照しない
  {
    files: [...frontendAppSourceFiles],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@www-template/api',
              message:
                'frontend presentation 層では API パッケージを直接 import せず、domain hooks を経由してください。',
            },
          ],
          patterns: [
            {
              group: ['@www-template/api/**'],
              message:
                'frontend presentation 層では API パッケージを直接 import せず、domain hooks を経由してください。',
            },
          ],
        },
      ],
    },
  },

  // web は公開面 LP なので domain / api に依存しない
  {
    files: [...frontendWebSourceFiles],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@www-template/api',
              message: 'web は公開面 WebPage なので API パッケージを import できません。',
            },
            {
              name: '@www-template/domain',
              message: 'web は公開面 WebPage なので domain パッケージを import できません。',
            },
          ],
          patterns: [
            {
              group: ['@www-template/api/**'],
              message: 'web は公開面 WebPage なので API パッケージを import できません。',
            },
            {
              group: ['@www-template/domain/**'],
              message: 'web は公開面 WebPage なので domain パッケージを import できません。',
            },
          ],
        },
      ],
    },
  },

  // packages 配下の index.ts は re-export 専用（実装禁止）
  {
    files: ['packages/**/index.ts'],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector:
            'Program > :not(ImportDeclaration):not(ExportNamedDeclaration):not(ExportAllDeclaration)',
          message: 'index.ts では実装を持たず、re-export のみにしてください。',
        },
        {
          selector: 'ExportNamedDeclaration[declaration]',
          message: 'index.ts では値や関数の実装を直接 export しないでください (re-export のみ)。',
        },
        {
          selector: 'ExportDefaultDeclaration',
          message: 'index.ts での default export は禁止です。re-export のみにしてください。',
        },
      ],
    },
  },

  // API SDK (生成コード) は厳格ルールを緩和
  {
    files: ['packages/frontend/api/src/generated/**/*.{ts,tsx}'],
    languageOptions: {
      parserOptions: {
        projectService: false,
        project: './packages/frontend/api/tsconfig.json',
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      '@typescript-eslint/consistent-type-definitions': 'off',
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-unsafe-argument': 'off',
      '@typescript-eslint/no-unsafe-member-access': 'off',
      '@typescript-eslint/no-unsafe-return': 'off',
      '@typescript-eslint/no-unsafe-call': 'off',
      '@typescript-eslint/no-unnecessary-condition': 'off',
      '@typescript-eslint/no-unnecessary-type-conversion': 'off',
      '@typescript-eslint/prefer-nullish-coalescing': 'off',
      '@typescript-eslint/strict-boolean-expressions': 'off',
      '@typescript-eslint/no-misused-spread': 'off',
      '@typescript-eslint/restrict-template-expressions': 'off',
      '@typescript-eslint/no-invalid-void-type': 'off',
      'eslint-comments/no-use': 'off',
      'eslint-comments/require-description': 'off',
      'import/order': 'off',
      'max-lines': 'off',
      'max-lines-per-function': 'off',
      'no-restricted-syntax': 'off',
      'unicorn/no-array-for-each': 'off',
    },
  },

  {
    files: frontendDomainSourceFiles,
    ignores: [...frontendDomainHookFiles, ...frontendDomainHookSvelteFiles],
    plugins: {
      'frontend-domain-purity': frontendDomainPurityPlugin,
    },
    rules: {
      'frontend-domain-purity/no-runtime-env': 'error',
    },
  },

  {
    files: frontendDomainSourceFiles,
    ignores: [...frontendDomainHookFiles, ...frontendDomainHookSvelteFiles],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@www-template/api',
              message:
                'frontend domain の API import は hooks/adapters に限定してください。純粋な domain module から SDK を参照しないでください。',
            },
          ],
          patterns: [
            {
              group: ['@www-template/api/**'],
              message:
                'frontend domain の API import は hooks/adapters に限定してください。純粋な domain module から SDK を参照しないでください。',
            },
          ],
        },
      ],
    },
  },

  // Domain 内 Feature 境界と相対パス制限
  {
    files: frontendDomainSourceFiles,
    rules: {
      'boundaries/element-types': [
        'error',
        {
          default: 'allow',
          message:
            'Domain feature boundary violation: %{from} is not allowed to import from %{target}.',
          rules: [
            {
              from: ['domain-status'],
              disallow: ['domain-auth', 'domain-observability'],
            },
            {
              from: ['domain-observability'],
              disallow: ['domain-auth', 'domain-status'],
            },
          ],
        },
      ],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['../../*', '../..'],
              message:
                'domain 内で ../../ を使わず、同 Feature 内の相対パスまたはパスエイリアスを使ってください。',
            },
          ],
        },
      ],
    },
  },

  // Domain composable/hook の命名規約
  {
    files: frontendDomainHookSvelteFiles,
    plugins: {
      'hooks-domain': {
        rules: {
          'require-domain-structure': {
            meta: {
              type: 'problem',
              docs: {
                description:
                  'Ensure hooks return both data/actions objects with Data/Actions types',
              },
              schema: [],
            },
            create(context) {
              const hookStack = [];
              const startHook = (node, typeInfo) => {
                hookStack.push({
                  node,
                  hasDomainResult: false,
                  hasType: typeInfo?.hasType ?? false,
                  typeIsValid: typeInfo?.typeIsValid ?? false,
                });
              };
              const endHook = (node) => {
                const info = hookStack.pop();
                if (!info) {
                  return;
                }
                if (!info.hasDomainResult) {
                  context.report({
                    node,
                    message:
                      'ドメイン概念の抽象化が不適切です。hooks は data/actions をまとめて返してください (ドメイン状態と操作の両方)。',
                  });
                }
                if (!info.hasType) {
                  context.report({
                    node,
                    message:
                      'ドメイン概念の抽象化が不適切です。hooks は戻り値に data/actions を含む型注釈（*Data / *Actions）を付けてください。',
                  });
                } else if (!info.typeIsValid) {
                  context.report({
                    node,
                    message:
                      'data の型は *Data、actions の型は *Actions で注釈してください（例: { data: FooData; actions: FooActions }）。',
                  });
                }
              };
              const currentHook = () => hookStack[hookStack.length - 1];

              const checkReturn = (arg) => {
                if (!arg) return false;
                if (arg.type === 'ObjectExpression') {
                  const hasData = arg.properties.some(
                    (prop) =>
                      prop.type === 'Property' &&
                      prop.key.type === 'Identifier' &&
                      prop.key.name === 'data'
                  );
                  const hasActions = arg.properties.some(
                    (prop) =>
                      prop.type === 'Property' &&
                      prop.key.type === 'Identifier' &&
                      prop.key.name === 'actions'
                  );
                  return hasData && hasActions;
                }
                return false;
              };

              const isHookName = (name) => /^use[\dA-Z].*/.test(name ?? '');

              const typeEndsWith = (typeName, suffix) =>
                typeof typeName === 'string' && typeName.endsWith(suffix);

              const getIdentifierName = (typeRef) => {
                if (typeRef.type === 'Identifier') return typeRef.name;
                if (typeRef.type === 'TSQualifiedName' && typeRef.right.type === 'Identifier') {
                  return typeRef.right.name;
                }
                return null;
              };

              const evaluateTypeLiteral = (typeNode) => {
                if (!typeNode || typeNode.type !== 'TSTypeLiteral') {
                  return { hasType: true, typeIsValid: false };
                }
                let dataOk = false;
                let actionsOk = false;
                for (const member of typeNode.members) {
                  if (
                    member.type === 'TSPropertySignature' &&
                    member.key.type === 'Identifier' &&
                    member.typeAnnotation
                  ) {
                    const name = member.key.name;
                    const typeAnn = member.typeAnnotation.typeAnnotation;
                    if (
                      name === 'data' &&
                      typeAnn.type === 'TSTypeReference' &&
                      typeEndsWith(getIdentifierName(typeAnn.typeName) ?? '', 'Data')
                    ) {
                      dataOk = true;
                    }
                    if (
                      name === 'actions' &&
                      typeAnn.type === 'TSTypeReference' &&
                      typeEndsWith(getIdentifierName(typeAnn.typeName) ?? '', 'Actions')
                    ) {
                      actionsOk = true;
                    }
                  }
                }
                return { hasType: true, typeIsValid: dataOk && actionsOk };
              };

              const evaluateTypeAnnotation = (tsAnnotation) => {
                if (!tsAnnotation) return { hasType: false, typeIsValid: false };
                const t =
                  tsAnnotation.type === 'TSTypeAnnotation'
                    ? tsAnnotation.typeAnnotation
                    : tsAnnotation;
                if (!t) return { hasType: false, typeIsValid: false };
                if (t.type === 'TSTypeLiteral') {
                  return evaluateTypeLiteral(t);
                }
                // Other shapes (type aliases etc.) are treated as present but not validated
                return { hasType: true, typeIsValid: false };
              };

              return {
                FunctionDeclaration(node) {
                  if (node.id && isHookName(node.id.name)) {
                    const typeInfo = evaluateTypeAnnotation(node.returnType);
                    startHook(node, typeInfo);
                  }
                },
                'FunctionDeclaration:exit'(node) {
                  if (node.id && isHookName(node.id.name)) {
                    endHook(node);
                  }
                },

                VariableDeclarator(node) {
                  if (
                    node.id.type === 'Identifier' &&
                    isHookName(node.id.name) &&
                    (node.init?.type === 'ArrowFunctionExpression' ||
                      node.init?.type === 'FunctionExpression')
                  ) {
                    const typeInfo =
                      evaluateTypeAnnotation(node.id.typeAnnotation) ||
                      evaluateTypeAnnotation(node.init.returnType);
                    startHook(node, typeInfo);
                    const body = node.init.body;
                    if (body && body.type !== 'BlockStatement' && checkReturn(body)) {
                      const info = currentHook();
                      if (info) info.hasDomainResult = true;
                    }
                  }
                },
                'VariableDeclarator:exit'(node) {
                  if (node.id.type === 'Identifier' && isHookName(node.id.name)) {
                    endHook(node);
                  }
                },

                ReturnStatement(node) {
                  const info = currentHook();
                  if (!info) return;
                  if (checkReturn(node.argument)) {
                    info.hasDomainResult = true;
                  }
                },
              };
            },
          },
        },
      },
    },
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector:
            'ExportNamedDeclaration > VariableDeclaration > VariableDeclarator[id.name!=/^(use[A-Z0-9].*)/]',
          message:
            'hooks ディレクトリでは値のエクスポートは use で始まるカスタムフックに限定してください。',
        },
        {
          selector:
            "ExportNamedDeclaration[exportKind!='type'] > FunctionDeclaration[id.name!=/^(use[A-Z0-9].*)/]",
          message:
            'hooks ディレクトリでは値のエクスポートは use で始まるカスタムフックに限定してください。',
        },
        {
          selector:
            "ExportNamedDeclaration[exportKind!='type'] > ExportSpecifier[exported.name!=/^(use[A-Z0-9].*)/]",
          message:
            'hooks ディレクトリから再エクスポートできる値は use で始まるカスタムフックのみです。',
        },
        {
          selector: 'ExportDefaultDeclaration > Identifier[name!=/^(use[A-Z0-9].*)/]',
          message:
            'hooks ディレクトリではデフォルトエクスポートも use で始まるカスタムフックにしてください。',
        },
        {
          selector: 'ExportDefaultDeclaration > FunctionDeclaration[id.name!=/^(use[A-Z0-9].*)/]',
          message:
            'hooks ディレクトリではデフォルトエクスポートも use で始まるカスタムフックにしてください。',
        },
        {
          selector: "ReturnStatement:has(Identifier[name='apiClient'])",
          message:
            'apiClient をそのまま返す/ラップするのは禁止です。hooks 内でドメインロジック・状態をまとめて返してください。',
        },
        {
          selector: "ExportSpecifier[exported.name='apiClient'], Identifier[name='apiClient']",
          message: 'apiClient の再エクスポートを禁止します。',
        },
        {
          selector:
            "ImportSpecifier[importKind='type']:not([parent.source.value='types']):not([parent.source.value^='types/'])",
          message:
            'hooks の型 import は src/types 経由にしてください (import type ... from "types")。',
        },
        {
          selector:
            "ImportDeclaration[importKind='type']:not([source.value='types']):not([source.value^='types/'])",
          message:
            'hooks の型 import は src/types 経由にしてください (import type ... from "types")。',
        },
        {
          selector: "CallExpression[callee.name='$state']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$state` は使えません。',
        },
        {
          selector: "CallExpression[callee.name='$derived']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$derived` は使えません。',
        },
        {
          selector: "CallExpression[callee.name='$effect']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$effect` は使えません。',
        },
        {
          selector: "CallExpression[callee.object.name='$effect'][callee.property.name='pre']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$effect.pre` は使えません。',
        },
        {
          selector: "CallExpression[callee.name='fetch']",
          message:
            'Pages, components, and hooks must call the shared apiClient instead of fetch directly.',
        },
        {
          selector: "CallExpression[callee.object.name='globalThis'][callee.property.name='fetch']",
          message:
            'Pages, components, and hooks must call the shared apiClient instead of fetch directly.',
        },
      ],
      'hooks-domain/require-domain-structure': 'error',
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@www-template/app',
              message: 'hooks では UI 層（app/pages/components）の import を禁止します。',
            },
            {
              name: '@www-template/ui',
              message: 'hooks では UI 層（ui/components）の import を禁止します。',
            },
            {
              name: 'axios',
              message: 'Use @www-template/api instead of axios.',
            },
            {
              name: 'cross-fetch',
              message: 'Use @www-template/api instead of performing manual fetches.',
            },
          ],
          patterns: [
            {
              group: [
                '@www-template/app/**',
                '@www-template/ui/**',
                '../app/**',
                '../../app/**',
                '../ui/**',
                '../../ui/**',
              ],
              message: 'hooks では UI 層（app/pages/components/ui）の import を禁止します。',
            },
          ],
        },
      ],
      'unicorn/filename-case': [
        'error',
        {
          case: 'camelCase',
        },
      ],
      'boundaries/element-types': [
        'error',
        {
          default: 'disallow',
          message: 'Clean Architecture violation: %{from} is not allowed to import from %{target}.',
          rules: [
            {
              from: ['frontend-domain'],
              allow: ['frontend-domain', 'frontend-api'],
            },
          ],
        },
      ],
    },
  },

  {
    files: frontendDomainHookSvelteFiles,
    rules: {
      'hooks-domain/require-domain-structure': 'error',
      'no-restricted-globals': ['error', 'window', 'document', 'localStorage'],
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@www-template/app',
              message: 'hooks では UI 層（app/pages/components）の import を禁止します。',
            },
            {
              name: '@www-template/ui',
              message: 'hooks では UI 層（ui/components）の import を禁止します。',
            },
            {
              name: 'axios',
              message: 'Use @www-template/api instead of axios.',
            },
            {
              name: 'cross-fetch',
              message: 'Use @www-template/api instead of performing manual fetches.',
            },
            {
              name: 'react',
              message: 'Svelte domain composable では React を import しないでください。',
            },
            {
              name: 'react-dom',
              message: 'Svelte domain composable では React DOM を import しないでください。',
            },
            {
              name: '@tanstack/react-query',
              message:
                'Svelte domain composable では React Query ではなく Svelte5 の state/composable へ移してください。',
            },
            {
              name: 'svelte/store',
              message:
                'Svelte5 の domain composable では `svelte/store` ではなく `$state` / `$derived` / `$effect` を使ってください。',
            },
            {
              name: 'svelte',
              importNames: [
                'onMount',
                'beforeUpdate',
                'afterUpdate',
                'tick',
                'setContext',
                'getContext',
              ],
              message:
                'Domain composable では lifecycle/context API を使わず、状態と副作用の集約だけに留めてください。',
            },
          ],
          patterns: [
            {
              group: [
                '@www-template/app/**',
                '@www-template/ui/**',
                '../app/**',
                '../../app/**',
                '../ui/**',
                '../../ui/**',
              ],
              message: 'hooks では UI 層（app/pages/components/ui）の import を禁止します。',
            },
            {
              group: ['**/*.svelte'],
              message: 'Domain composable から Svelte component を import しないでください。',
            },
          ],
        },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector:
            'ExportNamedDeclaration > VariableDeclaration > VariableDeclarator[id.name!=/^(use[A-Z0-9].*)/]',
          message:
            'hooks ディレクトリでは値のエクスポートは use で始まるカスタムフックに限定してください。',
        },
        {
          selector:
            "ExportNamedDeclaration[exportKind!='type'] > FunctionDeclaration[id.name!=/^(use[A-Z0-9].*)/]",
          message:
            'hooks ディレクトリでは値のエクスポートは use で始まるカスタムフックに限定してください。',
        },
        {
          selector:
            "ExportNamedDeclaration[exportKind!='type'] > ExportSpecifier[exported.name!=/^(use[A-Z0-9].*)/]",
          message:
            'hooks ディレクトリから再エクスポートできる値は use で始まるカスタムフックのみです。',
        },
        {
          selector: 'ExportDefaultDeclaration > Identifier[name!=/^(use[A-Z0-9].*)/]',
          message:
            'hooks ディレクトリではデフォルトエクスポートも use で始まるカスタムフックにしてください。',
        },
        {
          selector: 'ExportDefaultDeclaration > FunctionDeclaration[id.name!=/^(use[A-Z0-9].*)/]',
          message:
            'hooks ディレクトリではデフォルトエクスポートも use で始まるカスタムフックにしてください。',
        },
        {
          selector: "ReturnStatement:has(Identifier[name='apiClient'])",
          message:
            'apiClient をそのまま返す/ラップするのは禁止です。hooks 内でドメインロジック・状態をまとめて返してください。',
        },
        {
          selector: "ExportSpecifier[exported.name='apiClient'], Identifier[name='apiClient']",
          message: 'apiClient の再エクスポートを禁止します。',
        },
        {
          selector:
            "ImportSpecifier[importKind='type']:not([parent.source.value='types']):not([parent.source.value^='types/'])",
          message:
            'hooks の型 import は src/types 経由にしてください (import type ... from "types")。',
        },
        {
          selector:
            "ImportDeclaration[importKind='type']:not([source.value='types']):not([source.value^='types/'])",
          message:
            'hooks の型 import は src/types 経由にしてください (import type ... from "types")。',
        },
        {
          selector: "CallExpression[callee.name='fetch']",
          message:
            'Pages, components, and hooks must call the shared apiClient instead of fetch directly.',
        },
        {
          selector: "CallExpression[callee.object.name='globalThis'][callee.property.name='fetch']",
          message:
            'Pages, components, and hooks must call the shared apiClient instead of fetch directly.',
        },
        {
          selector:
            "MemberExpression[object.name='globalThis'][property.name=/^(window|document|localStorage|sessionStorage)$/]",
          message: 'Domain composable では DOM やブラウザストレージへ直接依存しないでください。',
        },
      ],
      'unicorn/filename-case': [
        'error',
        {
          case: 'camelCase',
        },
      ],
      'boundaries/element-types': [
        'error',
        {
          default: 'disallow',
          message: 'Clean Architecture violation: %{from} is not allowed to import from %{target}.',
          rules: [
            {
              from: ['frontend-domain'],
              allow: ['frontend-domain', 'frontend-api'],
            },
          ],
        },
      ],
    },
  },

  {
    files: frontendDomainPlainTsFiles,
    ignores: [
      'packages/frontend/domain/src/**/*.svelte.ts',
      'packages/frontend/domain/src/hooks/**/*.ts',
    ],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='$state']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$state` は使えません。',
        },
        {
          selector: "CallExpression[callee.name='$derived']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$derived` は使えません。',
        },
        {
          selector: "CallExpression[callee.name='$effect']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$effect` は使えません。',
        },
        {
          selector: "CallExpression[callee.object.name='$effect'][callee.property.name='pre']",
          message:
            'stateful な domain composable は `.svelte.ts` に配置してください。`.ts` で `$effect.pre` は使えません。',
        },
      ],
    },
  },

  // app / domain では直接 fetch しない（共通 API 経由）
  {
    files: [...frontendAppSourceFiles, ...frontendDomainSourceFiles],
    ignores: [
      'packages/frontend/app/src/**/*.test.ts',
      'packages/frontend/app/src/**/*.test.tsx',
      'packages/frontend/app/src/**/*.test.svelte',
      'packages/frontend/app/src/**/*.spec.ts',
      'packages/frontend/app/src/**/*.spec.tsx',
      'packages/frontend/app/src/**/*.spec.svelte',
      'packages/frontend/domain/src/**/*.test.ts',
      'packages/frontend/domain/src/**/*.test.tsx',
      'packages/frontend/domain/src/**/*.test.svelte.ts',
      'packages/frontend/domain/src/hooks/**/*.ts',
      'packages/frontend/domain/src/hooks/**/*.tsx',
      'packages/frontend/domain/src/hooks/**/*.svelte.ts',
      'packages/frontend/domain/src/hooks/**/*.svelte.js',
      'packages/frontend/domain/src/**/*.spec.ts',
      'packages/frontend/domain/src/**/*.spec.tsx',
      'packages/frontend/domain/src/**/*.spec.svelte.ts',
    ],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='fetch']",
          message:
            'Pages, components, and hooks must call the shared apiClient instead of fetch directly.',
        },
        {
          selector: "CallExpression[callee.object.name='globalThis'][callee.property.name='fetch']",
          message:
            'Pages, components, and hooks must call the shared apiClient instead of fetch directly.',
        },
      ],
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: 'axios',
              message: 'Use @www-template/api instead of axios.',
            },
            {
              name: 'cross-fetch',
              message: 'Use @www-template/api instead of performing manual fetches.',
            },
          ],
        },
      ],
    },
  },
  {
    files: [...frontendWebSourceFiles],
    ignores: [
      'packages/web/src/**/*.test.ts',
      'packages/web/src/**/*.test.tsx',
      'packages/web/src/**/*.test.svelte',
      'packages/web/src/**/*.spec.ts',
      'packages/web/src/**/*.spec.tsx',
      'packages/web/src/**/*.spec.svelte',
    ],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: 'axios',
              message: 'web では native fetch を使い、axios は使わないでください。',
            },
            {
              name: 'cross-fetch',
              message: 'web では native fetch を使い、cross-fetch は使わないでください。',
            },
          ],
        },
      ],
    },
  },
  {
    files: [...frontendRoutePageFiles, ...frontendComponentFiles],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@www-template/api',
              message: 'Pages/Components は Hooks 経由でAPIを呼び出してください。',
            },
            {
              name: '@www-template/domain',
              message: 'hooks は個別フックを指し示すパスで import してください。',
            },
          ],
          patterns: [
            {
              group: ['@www-template/app/src/components/**', '@www-template/web/src/components/**'],
              message: 'components 同士の循環参照を避け、必要なら hooks 経由にしてください。',
            },
          ],
        },
      ],
    },
  },
  {
    files: frontendNonReactSourceFiles,
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: 'react',
              message: 'frontend の active source では React を import しないでください。',
            },
            {
              name: 'react-dom',
              message: 'frontend の active source では React DOM を import しないでください。',
            },
            {
              name: '@tanstack/react-query',
              message:
                'frontend の active source では React Query を import しないでください。Svelte5 の domain composable へ寄せてください。',
            },
          ],
        },
      ],
    },
  },
  {
    files: [
      'packages/frontend/app/src/**/*.{tsx,jsx}',
      'packages/frontend/domain/src/**/*.{tsx,jsx}',
    ],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: 'Program',
          message:
            'packages/frontend/app と packages/frontend/domain では React/TSX ファイルを作らないでください。Svelte または TypeScript へ統一してください。',
        },
      ],
    },
  },
  {
    files: frontendAppRoutePageFiles,
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: 'svelte',
              importNames: ['onMount', 'beforeUpdate', 'afterUpdate', 'tick'],
              message:
                'route component の副作用は domain composable に委譲してください。Svelte lifecycle は route component では使わないでください。',
            },
          ],
        },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='$effect']",
          message:
            'route component の副作用は domain composable に集約してください。派生値は $derived を使い、I/O は domain に移してください。',
        },
        {
          selector: "CallExpression[callee.object.name='$effect'][callee.property.name='pre']",
          message:
            'route component の副作用は domain composable に集約してください。$effect.pre は route component では使用禁止です。',
        },
      ],
    },
  },
  {
    files: frontendAppComponentFiles,
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: 'svelte',
              importNames: ['onMount', 'beforeUpdate', 'afterUpdate', 'tick'],
              message:
                'frontend UI component の副作用は domain composable に委譲してください。Svelte lifecycle は `src/components` と `src/lib` では使わないでください。',
            },
          ],
        },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='$effect']",
          message:
            'frontend UI component の副作用は domain composable に集約してください。局所 UI state だけで済まない処理は domain に移してください。',
        },
        {
          selector: "CallExpression[callee.object.name='$effect'][callee.property.name='pre']",
          message:
            'frontend UI component の副作用は domain composable に集約してください。$effect.pre は `src/components` と `src/lib` では使用禁止です。',
        },
      ],
    },
  },
  {
    files: frontendAppSvelteKitImportFiles,
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-forbidden-imports': 'error',
    },
  },
  {
    files: frontendAppSvelteKitRouteModuleFiles,
    ignores: ['packages/frontend/app/src/routes/+layout.ts'],
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-export-names': [
        'error',
        {
          names: ['ssr', 'csr', 'prerender'],
          message:
            'frontend app の route mode は `src/routes/+layout.ts` だけで管理してください（`{{name}}` export 禁止）。',
        },
      ],
    },
  },
  {
    files: ['packages/frontend/app/src/routes/+layout.ts'],
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-export-names': [
        'error',
        {
          names: ['prerender'],
          message:
            'frontend app の root layout では `prerender` を export せず、`ssr = false` / `csr = true` だけで route mode を固定してください。',
        },
      ],
      'sveltekit-app-policy/require-auth-layout-mode': 'error',
    },
  },
  {
    files: frontendAppSvelteKitPageServerModuleFiles,
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-export-names': [
        'error',
        {
          names: ['actions'],
          message:
            'packages/web では SvelteKit の form action export（`{{name}}`）を禁止します。API は backend に集約してください。',
        },
      ],
    },
  },
  {
    files: frontendAppSvelteKitHookModuleFiles,
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-export-names': [
        'error',
        {
          names: ['handle', 'handleFetch'],
          message:
            'frontend app では SvelteKit の hook export（`{{name}}`）を禁止します。server 面は backend と SPA route 境界に集約してください。',
        },
      ],
    },
  },
  {
    files: frontendAppSvelteKitServerOnlyFiles,
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: 'Program',
          message:
            'packages/frontend/app では SvelteKit の server route / server hook / server-only lib を禁止します。API は backend に集約してください。',
        },
      ],
    },
  },
  {
    files: frontendWebSvelteKitImportFiles,
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-forbidden-imports': 'error',
    },
  },
  {
    files: frontendWebSvelteKitRouteModuleFiles,
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-export-names': [
        'error',
        {
          names: ['ssr'],
          message:
            '公開 route module では `ssr` export を禁止します。公開面は Cloudflare Workers 上の SvelteKit SSR default を維持してください。',
        },
      ],
    },
  },
  {
    files: frontendWebSvelteKitPageServerModuleFiles,
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-export-names': [
        'error',
        {
          names: ['actions'],
          message:
            'frontend app では SvelteKit の form action export（`{{name}}`）を禁止します。API は backend に集約してください。',
        },
      ],
    },
  },
  {
    files: frontendWebSvelteKitHookModuleFiles,
    plugins: {
      'sveltekit-app-policy': sveltekitAppPolicyPlugin,
    },
    rules: {
      'sveltekit-app-policy/no-export-names': [
        'error',
        {
          names: ['handle', 'handleFetch'],
          message:
            'packages/web では SvelteKit の hook export（`{{name}}`）を禁止します。server 面は backend に集約してください。',
        },
      ],
    },
  },
  {
    files: [
      'packages/web/src/routes/**/+server.{ts,js}',
      'packages/web/src/routes/**/+page.server.{ts,js}',
      'packages/web/src/routes/**/+layout.server.{ts,js}',
      'packages/web/src/hooks.server.{ts,js}',
      'packages/web/src/lib/server/**/*.{ts,js,svelte}',
      'packages/web/src/lib/server/**/*.svelte.{ts,js}',
    ],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: 'Program',
          message:
            'packages/web では SvelteKit の server route / server hook / server-only lib を禁止します。API は backend に集約してください。',
        },
      ],
    },
  },
  {
    files: [
      '**/*.test.ts',
      '**/*.test.tsx',
      '**/*.test.svelte',
      '**/*.test.svelte.ts',
      '**/*.spec.ts',
      '**/*.spec.tsx',
      '**/*.spec.svelte',
      '**/*.spec.svelte.ts',
    ],
    rules: {
      'no-restricted-imports': 'off',
      'no-restricted-syntax': 'off',
    },
  },
  // ESLint 設定ファイルやテストのゆるめ設定
  {
    files: ['eslint.config.js'],
    rules: {
      'import/extensions': 'off',
      'sonarjs/cognitive-complexity': 'off',
      'sonarjs/no-duplicate-string': 'off',
      'deprecation/deprecation': 'off',
    },
  },
  {
    files: [
      '**/*.test.ts',
      '**/*.test.tsx',
      '**/*.test.svelte',
      '**/*.test.svelte.ts',
      '**/*.spec.ts',
      '**/*.spec.tsx',
      '**/*.spec.svelte',
      '**/*.spec.svelte.ts',
    ],
    rules: {
      'sonarjs/no-duplicate-string': 'off',
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-unsafe-call': 'off',
      '@typescript-eslint/no-unsafe-member-access': 'off',
      '@typescript-eslint/no-unsafe-argument': 'off',
      '@typescript-eslint/no-unsafe-return': 'off',
      '@typescript-eslint/require-await': 'off',
    },
  },
  {
    files: [
      'packages/frontend/app/src/tests/**/*.{ts,tsx}',
      'packages/frontend/app/src/tests/**/*.svelte',
      'packages/web/src/tests/**/*.{ts,tsx}',
      'packages/web/src/tests/**/*.svelte',
    ],
    rules: {
      'no-restricted-imports': 'off',
      'no-restricted-syntax': 'off',
    },
  },

  // packages 全体で index.ts 経由の import を強制 + 行数制約（生成コード・テストは除外）
  {
    files: [
      'packages/**/src/**/*.{ts,tsx}',
      'packages/**/src/**/*.svelte.ts',
      'packages/**/src/**/*.svelte.js',
    ],
    ignores: [
      '**/index.ts',
      'packages/frontend/api/src/generated/**/*.{ts,tsx}',
      '**/*.test.ts',
      '**/*.test.tsx',
      '**/*.test.svelte.ts',
      '**/*.spec.ts',
      '**/*.spec.tsx',
      '**/*.spec.svelte.ts',
    ],
    rules: {
      'max-lines': [
        'error',
        {
          max: 500,
          skipComments: true,
          skipBlankLines: true,
        },
      ],
      'max-lines-per-function': [
        'error',
        {
          max: 100,
          skipComments: true,
          skipBlankLines: true,
          IIFEs: true,
        },
      ],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: [
                '**/src/**/!(*index)',
                '@www-template/**/!(*index)',
                './**/!(*index)',
                '../**/!(*index)',
              ],
              message: 'import は各ディレクトリの index.ts に統一してください。',
            },
          ],
        },
      ],
    },
  },
  {
    files: frontendUiSourceFiles,
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['../**'],
              message: '@ui エイリアスでパッケージ内の上位ディレクトリを参照してください。',
            },
            {
              group: [
                '**/src/**/!(*index)',
                '@www-template/**/!(*index)',
                './**/!(*index)',
                '../**/!(*index)',
              ],
              message: 'import は各ディレクトリの index.ts に統一してください。',
            },
          ],
        },
      ],
    },
  },
  // theme.ts は行数制約を緩和
  {
    files: ['**/theme.ts'],
    rules: {
      'max-lines': 'off',
      'max-lines-per-function': 'off',
    },
  },
  // JavaScript ファイルの設定
  {
    files: ['**/*.js', '**/*.cjs', '**/*.mjs'],
    ...tseslint.configs.disableTypeChecked,
  },
  // vitest config は型情報なしで lint
  {
    files: ['packages/frontend/ui/vitest.config.ts'],
    ...tseslint.configs.disableTypeChecked,
  },
  // Storybook 関連は型情報なしで lint
  {
    files: [
      'packages/frontend/ui/.storybook/**/*.{ts,tsx}',
      'packages/frontend/ui/src/**/*.stories.ts',
      'packages/frontend/ui/src/**/*.stories.tsx',
    ],
    ...tseslint.configs.disableTypeChecked,
  },

  {
    files: frontendSvelteFiles,
    plugins: {
      'frontend-css-policy': frontendCssPolicyPlugin,
    },
    rules: {
      'frontend-css-policy/no-svelte-style-tag': 'error',
      'frontend-css-policy/no-tailwind-arbitrary-values': 'error',
    },
  },

  // 無視するファイル
  {
    ignores: [
      '**/node_modules/**',
      '**/dist/**',
      '**/build/**',
      '**/storybook-static/**',
      '**/.wrangler/**',
      '**/.mcp/**',
      '**/.opencode/**',
      '**/.serena/**',
      'packages/typespec/openapi/**',
      'packages/typespec/tsp-output/**',
      '**/pnpm-lock.yaml',
    ],
  }
);

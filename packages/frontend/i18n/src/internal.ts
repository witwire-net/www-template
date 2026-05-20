/**
 * i18n runtime 内部で使う共通ユーティリティです。
 *
 * ここに置く処理は package 外へ公開せず、catalog の検証・参照・補間など
 * 複数モジュールで共有する低レベル処理だけに限定します。
 */
export type MaybePromise<T> = T | Promise<T>;

/**
 * 翻訳文字列の補間に渡す値の型です。
 *
 * 文字列置換だけに絞ることで、JSON catalog と typed translator の
 * 責務を明確に保ちます。
 */
export type TranslationValue = string | number | boolean;

const prohibitedKeys = new Set(['__proto__', 'prototype', 'constructor']);

const createRecord = <T>(): Record<string, T> => Object.create(null) as Record<string, T>;

/**
 * 値が安全な plain object かどうかを判定します。
 *
 * 配列、null、関数、class instance を拒否し、JSON catalog の入力面を
 * 予測可能な形に固定します。
 */
export function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  const prototype = Reflect.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

/**
 * catalog tree を再帰的に検証し、安全な null-prototype object へ複製します。
 *
 * 各キーは空文字列、`.` を含む文字列、または prototype 汚染に使われるキーを
 * 拒否します。leaf は文字列のみを許可し、翻訳値以外の混入を止めます。
 */
export function cloneCatalogTree(
  source: unknown,
  path: readonly string[] = []
): Record<string, unknown> {
  if (!isPlainObject(source)) {
    const location = path.length === 0 ? '<root>' : path.join('.');
    throw new TypeError(`i18n catalog は plain object でなければなりません: ${location}`);
  }

  const output = createRecord<unknown>();

  for (const [key, value] of Object.entries(source)) {
    const nextPath = [...path, key];

    if (key.trim().length === 0) {
      throw new Error(`i18n catalog のキーは空にできません: ${nextPath.join('.')}`);
    }

    if (key.includes('.')) {
      throw new Error(`i18n catalog のキーに "." は使えません: ${nextPath.join('.')}`);
    }

    if (prohibitedKeys.has(key)) {
      throw new Error(`i18n catalog で禁止されたキーです: ${nextPath.join('.')}`);
    }

    if (typeof value === 'string') {
      Reflect.set(output, key, value);
      continue;
    }

    if (isPlainObject(value)) {
      Reflect.set(output, key, cloneCatalogTree(value, nextPath));
      continue;
    }

    throw new Error(
      `i18n catalog の値は string または plain object でなければなりません: ${nextPath.join('.')}`
    );
  }

  return Object.freeze(output);
}

/**
 * catalog tree の leaf 文字列パスを収集します。
 *
 * 返り値は重複のない dot 区切りパスで、coverage と translator の参照に使います。
 */
export function collectCatalogLeafPaths(tree: Record<string, unknown>, prefix = ''): string[] {
  const paths: string[] = [];

  for (const [key, value] of Object.entries(tree)) {
    const nextPath = prefix.length === 0 ? key : `${prefix}.${key}`;

    if (typeof value === 'string') {
      paths.push(nextPath);
      continue;
    }

    if (isPlainObject(value)) {
      paths.push(...collectCatalogLeafPaths(value, nextPath));
    }
  }

  return paths;
}

/**
 * 指定した dot 区切りパスで catalog leaf を探します。
 *
 * 見つからなかった場合は `undefined` を返し、呼び出し側で fallback を
 * 適用できるようにします。
 */
export function getCatalogLeaf(
  tree: Record<string, unknown>,
  keyPath: string,
  separator: string
): string | undefined {
  const segments = keyPath.split(separator).filter((segment) => segment !== '');
  if (segments.length === 0) {
    return undefined;
  }

  let current: unknown = tree;

  for (const segment of segments) {
    if (!isPlainObject(current)) {
      return undefined;
    }

    const record: Record<string, unknown> = current;
    current = Reflect.get(record, segment);
  }

  return typeof current === 'string' ? current : undefined;
}

/**
 * 文字列テンプレートの `{name}` 形式を補間します。
 *
 * 渡された値が不足している場合は失敗させ、翻訳文中のプレースホルダーを
 * 放置しないようにします。
 */
export function interpolateTemplate(
  template: string,
  values: Readonly<Record<string, TranslationValue>> = {}
): string {
  return template.replace(/{(\w+)}/g, (_match, key: string) => {
    if (!Object.prototype.hasOwnProperty.call(values, key)) {
      throw new Error(`i18n template の補間値が不足しています: ${key}`);
    }

    return String(Reflect.get(values, key));
  });
}

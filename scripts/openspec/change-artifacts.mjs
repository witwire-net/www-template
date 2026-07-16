import { readdirSync, statSync } from 'node:fs';
import path from 'node:path';

/**
 * 指定したパスが存在するディレクトリかを安全に判定する。
 *
 * @param {string} absolutePath - 判定対象の絶対パス。
 * @returns {boolean} ディレクトリとして存在する場合は `true`。
 */
export function isDirectory(absolutePath) {
  try {
    return statSync(absolutePath).isDirectory();
  } catch {
    return false;
  }
}

/**
 * 未アーカイブ Change 配下から条件に一致する artifact を再帰的に収集する。
 *
 * archived Change は現在の workflow と lint の対象外です。この関数は artifact
 * 内容を解釈せず、呼び出し側が与える判定関数に従ってパスだけを返します。
 *
 * @param {string} repositoryRoot - リポジトリ root の絶対パス。
 * @param {(absolutePath: string, fileName: string) => boolean} includeArtifact - artifact を含める条件。
 * @returns {string[]} 条件に一致した artifact の絶対パス。
 */
export function collectActiveChangeArtifacts(repositoryRoot, includeArtifact) {
  const changesDirectory = path.join(repositoryRoot, 'openspec', 'changes');
  if (!isDirectory(changesDirectory)) return [];

  /** @type {string[]} */
  const artifacts = [];
  /** @type {string[]} */
  const pendingDirectories = [changesDirectory];

  while (pendingDirectories.length > 0) {
    const currentDirectory = pendingDirectories.pop();
    if (!currentDirectory) continue;

    for (const entry of readdirSync(currentDirectory, { withFileTypes: true })) {
      const entryPath = path.join(currentDirectory, entry.name);
      if (entry.isDirectory()) {
        // archive は確定済みの履歴なので、現在の Change に対する guard から除外します。
        if (entry.name !== 'archive') pendingDirectories.push(entryPath);
        continue;
      }

      if (entry.isFile() && includeArtifact(entryPath, entry.name)) artifacts.push(entryPath);
    }
  }

  return artifacts;
}

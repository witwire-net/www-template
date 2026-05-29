import { mkdir, readdir, rename, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

// TypeSpec package の root を script の場所から解決し、呼び出し元の cwd に依存しない生成後処理にする。
const packageRoot = fileURLToPath(new URL('..', import.meta.url));

// OpenAPI emitter が surface 名付きで出力した中間 artifact を、この repository で追跡する安定ファイル名へ正規化する。
const openApiDirectory = join(packageRoot, 'openapi');

// Product と Admin の service 名を TypeSpec namespace に合わせ、既存の Product 生成物パスと新しい Admin 生成物パスを固定する。
const artifactMappings = [
  {
    sourceFileName: 'WWWTemplate.openapi.json',
    destinationFileName: 'openapi.json',
  },
  {
    sourceFileName: 'Admin.openapi.json',
    destinationFileName: 'admin.openapi.json',
  },
];

// 生成先 directory が存在しない初回実行でも、後続の rename が directory 不在で失敗しないようにする。
await mkdir(openApiDirectory, { recursive: true });

for (const artifactMapping of artifactMappings) {
  // TypeSpec emitter の出力名と repository が公開する安定名を、それぞれ絶対 path として組み立てる。
  const sourcePath = join(openApiDirectory, artifactMapping.sourceFileName);
  const destinationPath = join(openApiDirectory, artifactMapping.destinationFileName);

  // 既存の安定名 artifact は古い生成結果なので、rename 前に削除して同一 filesystem 上の置換を明示する。
  await rm(destinationPath, { force: true });

  try {
    // 中間 artifact を安定名へ移動し、Product/Admin の出力が混ざらない形で追跡対象にする。
    await rename(sourcePath, destinationPath);
  } catch (error) {
    // service 名の変更や emitter 設定の失敗を検出しやすくするため、実際に出力された file 一覧を添えて失敗させる。
    const generatedFiles = await readdir(openApiDirectory);
    throw new Error(
      `OpenAPI artifact normalization failed for ${artifactMapping.sourceFileName}. ` +
        `Generated files: ${generatedFiles.join(', ')}`,
      { cause: error },
    );
  }
}

import { defineConfig } from 'orval';

// Admin OpenAPI だけを入力にして、Product SDK の出力先へ書き込まないようにする。
// 入力: packages/typespec/openapi/admin.openapi.json。
// 出力: packages/admin/api/src/generated/client.ts。
// 副作用: pnpm gen 実行時に Admin SDK 生成物を Admin package 配下だけへ更新する。
export default defineConfig({
  sdk: {
    input: '../../typespec/openapi/admin.openapi.json',
    output: {
      target: './src/generated/client.ts',
      client: 'fetch',
      baseUrl: '',
      clean: true,
    },
  },
});

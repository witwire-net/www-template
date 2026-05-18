// SvelteKit アプリケーションの型定義
// App.Locals: サーバーサイドリクエスト毎に設定される認証済みオペレーター情報
// App.Platform: ランタイム環境変数

declare global {
  namespace App {
    interface Locals {
      /**
       * 認証済みオペレーター情報。
       * hooks.server.ts で JWT cookie → Valkey active session → DB current role の検証を経て設定される。
       * 未認証時は null。
       */
      operator: { id: string; email: string; role: string; sessionId: string; jti: string } | null;
    }

    interface Platform {
      /**
       * ランタイム環境変数のレコード。
       * adapter-node 実行時や Docker コンテナ内で注入される。
       */
      env: Record<string, string>;
    }
  }
}

export {};

declare module 'bcryptjs';

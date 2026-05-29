// SvelteKit アプリケーションの型定義。
// Admin Console は静的 SPA として配信し、認証・CSRF・永続化は Go Admin API が所有する。

declare global {
  namespace App {
    // server-only Locals / Platform を宣言しないことで、SvelteKit server runtime への依存を型境界からも排除する。
  }
}

export {};

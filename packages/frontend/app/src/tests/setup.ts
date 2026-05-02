import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeAll, vi } from 'vitest';

import { resetMockData } from './mocks/handlers';
import { server } from './mocks/server';

// jsdom では window.matchMedia が未実装。
// Svelte UI コンポーネントの media-query / motion 依存を満たすためグローバルモックを設定。
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// MSW サーバーの起動・停止
beforeAll(() => {
  server.listen({ onUnhandledRequest: 'warn' });
});

afterEach(() => {
  // MSW のデータを初期状態に戻す
  resetMockData();
  // MSW ハンドラーをリセット
  server.resetHandlers();
});

afterAll(() => {
  server.close();
});

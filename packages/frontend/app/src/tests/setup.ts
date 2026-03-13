import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeAll } from 'vitest';

import { resetMockData } from './mocks/handlers';
import { server } from './mocks/server';

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

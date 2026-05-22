import { describe, expect, it } from 'vitest';

import { load } from './+page.server.js';

describe('operator setup page server contract', () => {
  it('登録済み session を持つ operator は setup token 画面へ入れず Dashboard に戻る', async () => {
    await expect(
      Promise.resolve().then(() => load({ locals: { operator: authedOperator() } } as never))
    ).rejects.toMatchObject({
      status: 303,
      location: '/',
    });
    await expect(Promise.resolve(load({ locals: { operator: null } } as never))).resolves.toEqual(
      {}
    );
  });
});

function authedOperator(): NonNullable<App.Locals['operator']> {
  return {
    id: 'op-1',
    email: 'admin@example.test',
    role: 'operator',
    locale: 'ja',
    sessionId: 'sess-1',
    jti: 'jti-1',
  };
}

import { describe, expect, it } from 'vitest';

import { createJsonCatalogLoader, loadJsonCatalog } from './catalog';

describe('catalog', () => {
  it('JSON catalog を安全な tree に変換する', () => {
    const catalog = loadJsonCatalog({
      auth: {
        login: 'Log in',
        heading: 'Welcome, {name}',
      },
    });

    expect(catalog.auth.login).toBe('Log in');
    expect(catalog.auth.heading).toBe('Welcome, {name}');
  });

  it('不正な catalog を拒否する', () => {
    expect(() => loadJsonCatalog({ '.': 'bad' })).toThrowError(/キーに "."/u);
    expect(() => loadJsonCatalog({ auth: ['bad'] })).toThrowError(/plain object/iu);
  });

  it('locale ごとの namespace loader を束ねる', async () => {
    interface TestNamespace {
      common: { greeting: string };
    }
    const loader = createJsonCatalogLoader<TestNamespace>({
      ja: {
        common: () => loadJsonCatalog({ greeting: 'こんにちは' }),
      },
      en: {
        common: () => loadJsonCatalog({ greeting: 'Hello' }),
      },
    });

    await expect(loader.load('ja')).resolves.toEqual({ common: { greeting: 'こんにちは' } });
    await expect(loader.loadAll()).resolves.toEqual({
      ja: { common: { greeting: 'こんにちは' } },
      en: { common: { greeting: 'Hello' } },
    });
  });
});

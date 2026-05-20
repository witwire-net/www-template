import { describe, expect, it } from 'vitest';

import { loadJsonCatalog } from './catalog';
import { assertCatalogCoverage, getCatalogCoverage } from './coverage';

describe('coverage', () => {
  it('locale 間の辞書差分を検出する', () => {
    interface TestNamespace {
      common: { greeting: string; farewell?: string };
    }
    const catalogs: Record<'ja' | 'en', TestNamespace> = {
      ja: {
        common: loadJsonCatalog({
          greeting: 'こんにちは',
          farewell: 'さようなら',
        }),
      },
      en: {
        common: loadJsonCatalog({
          greeting: 'Hello',
        }),
      },
    };

    const report = getCatalogCoverage(catalogs);

    expect(report.complete).toBe(false);
    expect(report.issues).toEqual([
      {
        locale: 'en',
        namespace: 'common',
        missingKeys: ['farewell'],
      },
    ]);
    expect(() => {
      assertCatalogCoverage(catalogs);
    }).toThrowError(/coverage が不完全/iu);
  });
});

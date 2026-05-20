import { describe, expect, it } from 'vitest';

import { loadJsonCatalog } from './catalog';
import { defineI18nConfig } from './config';
import { createTranslator } from './translator';

describe('translator', () => {
  interface TestNamespace {
    common: { greeting: string; goodbye: string };
  }
  const catalogs: Record<'ja' | 'en', TestNamespace> = {
    ja: {
      common: loadJsonCatalog({
        greeting: 'こんにちは、{name}さん',
        goodbye: 'またね',
      }),
    },
    en: {
      common: loadJsonCatalog({
        greeting: 'Hello, {name}',
        goodbye: 'See you',
      }),
    },
  };

  it('typed translator が locale に応じて翻訳する', () => {
    const translator = createTranslator<TestNamespace>(
      defineI18nConfig({ locale: 'en-US' }),
      catalogs
    );

    expect(translator.locale).toBe('en');
    expect(translator.t('common', 'greeting', { name: 'Ada' })).toBe('Hello, Ada');
    expect(translator.t('common', 'goodbye')).toBe('See you');
    expect(translator.has('common', 'goodbye')).toBe(true);
  });

  it('fallback locale を使って翻訳を補完する', () => {
    const translator = createTranslator<TestNamespace>(
      defineI18nConfig({ locale: 'fr', fallbackLocale: 'en' }),
      catalogs
    );

    expect(translator.locale).toBe('ja');
    expect(translator.t('common', 'greeting', { name: 'Taro' })).toBe('こんにちは、Taroさん');
  });

  it('欠落した key は失敗させる', () => {
    const translator = createTranslator<TestNamespace>(
      defineI18nConfig({ locale: 'ja' }),
      catalogs
    );

    expect(() => translator.t('common', 'missing' as never)).toThrowError(/key が見つかりません/iu);
  });
});

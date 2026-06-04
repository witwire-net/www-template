import { describe, expect, it } from 'vitest';

import { extractLocaleFromPath, resolveLocale, useI18n } from '../../lib/i18n';

describe('[LOCALIZATION-FE-S001] web i18n locale 抽出と解決', () => {
  it('extractLocaleFromPath: 対応ロケールを URL path から抽出する', () => {
    expect(extractLocaleFromPath('/ja')).toBe('ja');
    expect(extractLocaleFromPath('/en')).toBe('en');
    expect(extractLocaleFromPath('/ja/about')).toBe('ja');
  });

  it('extractLocaleFromPath: 未対応ロケールは null を返す', () => {
    expect(extractLocaleFromPath('/fr')).toBeNull();
    expect(extractLocaleFromPath('/')).toBeNull();
    expect(extractLocaleFromPath('')).toBeNull();
  });

  it('resolveLocale: browser 言語から対応ロケールを解決する', () => {
    expect(resolveLocale(['ja-JP', 'en-US'])).toBe('ja');
    expect(resolveLocale(['en-US', 'ja-JP'])).toBe('en');
    expect(resolveLocale(['fr-FR'])).toBe('ja'); // fallback
  });
});

describe('[LOCALIZATION-FE-S002] web translator 生成', () => {
  it('useI18n: ja locale の translator を生成する', () => {
    const i18n = useI18n('ja');
    expect(i18n.locale).toBe('ja');
    expect(i18n.t('common.home')).toBe('Home');
    expect(i18n.t('common.heroTitle')).toBe('公開面と認証面を、再利用しやすい層として組み立てる。');
  });

  it('useI18n: en locale の translator を生成する', () => {
    const i18n = useI18n('en');
    expect(i18n.locale).toBe('en');
    expect(i18n.t('common.home')).toBe('Home');
    expect(i18n.t('common.heroCtaPrimary')).toBe('Try login');
  });
});

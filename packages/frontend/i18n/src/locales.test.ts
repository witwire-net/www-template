import { describe, expect, it } from 'vitest';

import {
  DEFAULT_LOCALE,
  FALLBACK_LOCALE,
  SUPPORTED_LOCALES,
  isLocale,
  normalizeLocale,
  resolveLocale,
} from './locales';

describe('locales', () => {
  it('対応ロケールを固定する', () => {
    expect(SUPPORTED_LOCALES).toEqual(['ja', 'en']);
    expect(DEFAULT_LOCALE).toBe('ja');
    expect(FALLBACK_LOCALE).toBe('ja');
  });

  it('locale tag を正規化する', () => {
    expect(normalizeLocale('JA-JP')).toBe('ja');
    expect(normalizeLocale('en_us')).toBe('en');
    expect(normalizeLocale('fr')).toBeNull();
  });

  it('候補から locale を決定する', () => {
    expect(resolveLocale(['fr', 'en-US'])).toBe('en');
    expect(resolveLocale(undefined)).toBe('ja');
    expect(isLocale('ja')).toBe(true);
    expect(isLocale('fr')).toBe(false);
  });
});

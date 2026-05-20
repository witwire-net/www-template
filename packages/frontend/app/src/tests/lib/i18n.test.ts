import { afterEach, describe, expect, it } from 'vitest';

import { resolveUnauthenticatedLocale, useI18n } from '../../lib/i18n';

describe('[LOCALIZATION-FE-S006] app i18n locale resolver と dot-key translator', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('useI18n: dot-delimited key と interpolation を返す', () => {
    const i18n = useI18n('en');

    expect(i18n.locale).toBe('en');
    expect(i18n.t('login.login')).toBe('Login');
    expect(i18n.t('device-manager.logoutButtonAriaLabel', { deviceName: 'Safari on iOS' })).toBe(
      'Logout Safari on iOS'
    );
  });

  it('resolveUnauthenticatedLocale: 保存済み locale を優先する', () => {
    localStorage.setItem('www-template:locale', 'en');

    expect(resolveUnauthenticatedLocale()).toBe('en');
  });
});

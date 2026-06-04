import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

import { useI18n } from '../../lib/i18n';
import { load as rootLoad } from '../../routes/+page.server';
import { load as localeLoad } from '../../routes/[locale]/+page';

const routesRoot = join(process.cwd(), 'src/routes');

/** src 配下の任意ファイルを読み、固定 lang 属性など route 外の雛形も検証できるようにする。 */
function readSource(pathname: string): string {
  return readFileSync(join(process.cwd(), 'src', pathname), 'utf8');
}

function readRoute(pathname: string): string {
  return readFileSync(join(routesRoot, pathname), 'utf8');
}

function expectRouteContains(source: string, snippets: string[]): void {
  for (const snippet of snippets) {
    expect(source).toContain(snippet);
  }
}

describe('[LOCALIZATION-FE-S003] web route の root redirect / supported locale / unsupported locale を検証する', () => {
  it('root `/` は request header の Accept-Language から対応 locale へ redirect する', () => {
    try {
      rootLoad({
        request: new Request('https://www-template.test/?from=home', {
          headers: {
            'accept-language': 'en-US,en;q=0.9,ja;q=0.8',
          },
        }),
        url: new URL('https://www-template.test/?from=home'),
      });
      throw new Error('redirect が throw される想定でした。');
    } catch (error) {
      expect(error).toMatchObject({ status: 302, location: '/en?from=home' });
    }
  });

  it('root `/` は未対応の Accept-Language でも既定 locale へ redirect する', () => {
    try {
      rootLoad({
        request: new Request('https://www-template.test/?from=home', {
          headers: {
            'accept-language': 'fr-FR,fr;q=0.9',
          },
        }),
        url: new URL('https://www-template.test/?from=home'),
      });
      throw new Error('redirect が throw される想定でした。');
    } catch (error) {
      expect(error).toMatchObject({ status: 302, location: '/ja?from=home' });
    }
  });

  it('supported locale の route source は locale catalog key を参照する', () => {
    const localePageSource = readRoute('[locale]/+page.svelte');
    expectRouteContains(localePageSource, [
      "i18n.t('common.heroTitle')",
      "i18n.t('common.heroLead')",
      "i18n.t('common.heroCtaPrimary')",
    ]);

    const layoutSource = readRoute('+layout.svelte');
    expectRouteContains(layoutSource, ["aria-label={i18n.t('common.languageSwitchAriaLabel')}"]);

    const jaI18n = useI18n('ja');
    const enI18n = useI18n('en');
    expect(jaI18n.t('common.languageSwitchAriaLabel')).toBe('言語切り替え');
    expect(enI18n.t('common.languageSwitchAriaLabel')).toBe('Language switch');
  });

  it('unsupported locale path は 404 を返す', () => {
    try {
      localeLoad({ url: new URL('https://www-template.test/fr') });
      throw new Error('404 が throw される想定でした。');
    } catch (error) {
      expect(error).toMatchObject({ status: 404 });
    }
  });

  it('locale page は document.documentElement.lang を routed locale に設定する', () => {
    const localePageSource = readRoute('[locale]/+page.svelte');
    expect(localePageSource).toContain('document.documentElement.lang');
    expect(localePageSource).toContain('document.documentElement.lang = locale;');
  });

  it('web app.html は単一 locale の lang 属性を固定しない', () => {
    const appHtmlSource = readSource('app.html');
    expect(appHtmlSource).toContain('<html>');
    expect(appHtmlSource).not.toMatch(/<html[^>]*\slang=/u);
  });
});

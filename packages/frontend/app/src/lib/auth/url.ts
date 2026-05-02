/**
 * 指定したクエリパラメータを現在の URL から除去する。
 * パラメータが存在した場合は window.history.replaceState でブラウザの可視 URL から
 * 即時除去し、browser history / Referer / 画面共有経由の secret leakage を防ぐ。
 *
 * @param paramName - 除去対象のクエリパラメータ名。
 * @returns パラメータが存在した場合はその値、存在しなければ null。
 */
export function removeQueryParamFromUrl(paramName: string): string | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const params = new URLSearchParams(window.location.search);
  const value = params.get(paramName);

  if (value !== null) {
    window.history.replaceState({}, document.title, window.location.pathname);
  }

  return value;
}

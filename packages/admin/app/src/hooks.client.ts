import { configureAdminContextIndexStorage } from '@www-template/admin-domain';

// Admin client boot 用の最小フックファイル。
// 認証 API と cache 制御は Go Admin API と domain layer へ寄せつつ、browser storage の実体だけを app 層で注入する。

const adminStorageListenerMap = new Map<
  (event: { key: string | null }) => void,
  (event: StorageEvent) => void
>();

// Admin domain は DOM global へ直接依存しないため、origin-local localStorage と storage event を client hook から渡す。
configureAdminContextIndexStorage(globalThis.localStorage, {
  addStorageListener: (listener) => {
    // 同一 Admin origin の別 tab で context index が変わった場合に domain の購読 callback へ転送する。
    const browserListener = (event: StorageEvent) => {
      // domain port へ渡す情報を key だけに絞り、StorageEvent 全体を domain 境界へ漏らさない。
      listener({ key: event.key });
    };
    adminStorageListenerMap.set(listener, browserListener);
    globalThis.window.addEventListener('storage', browserListener);
  },
  removeStorageListener: (listener) => {
    // route/component 破棄時に storage listener を外し、古い auth state 反映を防ぐ。
    const browserListener = adminStorageListenerMap.get(listener);
    if (browserListener === undefined) return;
    globalThis.window.removeEventListener('storage', browserListener);
    adminStorageListenerMap.delete(listener);
  },
});

export {};

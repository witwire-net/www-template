import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

const passkeysPageSource = readFileSync(
  fileURLToPath(new URL('./+page.svelte', import.meta.url)),
  'utf8'
);

describe('passkey management page source contract', () => {
  it('登録済み passkey の安全な identifier と一覧 metadata を画面に描画する', () => {
    // Svelte component は node 環境の Admin Vitest で直接 mount しないため、一覧描画の契約を source 上で固定する。
    expect(passkeysPageSource).toContain('{#each passkeys as passkey, index (passkey.id)}');
    expect(passkeysPageSource).toContain('Passkey {index + 1}');
    expect(passkeysPageSource).toContain('Credential ID: {passkey.id}');
    expect(passkeysPageSource).toContain('登録日時: {formatDate(passkey.createdAt)}');
    expect(passkeysPageSource).toContain(
      "バックアップ: {passkey.backupEligible ? (passkey.backupState ? '同期済み' : '対応') : '端末固定'}"
    );
  });

  it('新しい passkey 追加は WebAuthn 登録後に最新一覧へ差し替える', () => {
    // 追加 flow が start → browser WebAuthn → finish → refresh の順で一覧を更新することを確認する。
    expect(passkeysPageSource).toContain("globalThis.fetch('/api/admin/auth/passkeys/start'");
    expect(passkeysPageSource).toContain(
      'const attestation = await startRegistration(startPayload.options);'
    );
    expect(passkeysPageSource).toContain("globalThis.fetch('/api/admin/auth/passkeys/finish'");
    expect(passkeysPageSource).toContain('await refreshPasskeys();');
    expect(passkeysPageSource).toContain("message = 'この端末のパスキーを追加しました。';");
  });

  it('最後の 1 件は削除不可で、2 件以上は確認 dialog 経由で削除できる', () => {
    // ロックアウト防止の disabled 条件と、2 件以上のときだけ進む削除確認 path を同時に固定する。
    expect(passkeysPageSource).toContain(
      'disabled={passkeys.length <= 1 || deletingId === passkey.id}'
    );
    expect(passkeysPageSource).toContain('onclick={() => { requestDelete(passkey.id); }}');
    expect(passkeysPageSource).toContain('onConfirm={confirmDelete}');
    expect(passkeysPageSource).toContain("method: 'DELETE'");
  });

  it('WebAuthn 登録キャンセル時は一覧を更新せず再試行メッセージを表示する', () => {
    // catch 節では refreshPasskeys を呼ばず、登録前の passkeys state を維持する契約を確認する。
    const catchStart = passkeysPageSource.indexOf('} catch {');
    const finallyStart = passkeysPageSource.indexOf('} finally {', catchStart);
    const catchBlock = passkeysPageSource.slice(catchStart, finallyStart);
    expect(catchBlock).toContain(
      "message = 'パスキーを追加できませんでした。認証状態を確認して再試行してください。';"
    );
    expect(catchBlock).not.toContain('refreshPasskeys');
  });
});

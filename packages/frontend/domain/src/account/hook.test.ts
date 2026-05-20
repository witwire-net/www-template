import { describe, expect, it, vi } from 'vitest';

import { accountApi } from '@www-template/api';

import { useAccount } from './hook.svelte';

describe('useAccount hook', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    const { actions } = useAccount();
    actions.reset();
  });

  it('[LOCALIZATION-FE-S004] loadAccountSetting: 成功時に state.account.setting.locale が更新される', async () => {
    const { data, actions } = useAccount();

    vi.spyOn(accountApi, 'getSettings').mockResolvedValue({
      setting: { locale: 'en' },
    } as Awaited<ReturnType<typeof accountApi.getSettings>>);

    await actions.loadAccountSetting({ Authorization: 'Bearer test' });

    expect(data.state.account).not.toBeNull();
    expect(data.state.account?.setting.locale).toBe('en');
    expect(data.state.loading).toBe(false);
    expect(data.state.error).toBeNull();
  });

  it('[LOCALIZATION-FE-S004] loadAccountSetting: 失敗時に state.error が設定される', async () => {
    const { data, actions } = useAccount();

    vi.spyOn(accountApi, 'getSettings').mockRejectedValue(new Error('network error'));

    await actions.loadAccountSetting({ Authorization: 'Bearer test' });

    expect(data.state.account).toBeNull();
    expect(data.state.loading).toBe(false);
    expect(data.state.error).toBe('account-settings-load-failed');
  });

  it('[LOCALIZATION-FE-S005] updateLocale: 成功時に state.account.setting.locale が更新される', async () => {
    const { data, actions } = useAccount();

    // 事前に account state を設定
    actions.applySnapshot('account-123', 'ja');
    expect(data.state.account?.setting.locale).toBe('ja');

    vi.spyOn(accountApi, 'updateSettings').mockResolvedValue({
      setting: { locale: 'en' },
    } as Awaited<ReturnType<typeof accountApi.updateSettings>>);

    const result = await actions.updateLocale('en', { Authorization: 'Bearer test' });

    expect(result).toBe(true);
    expect(data.state.account?.setting.locale).toBe('en');
    expect(data.state.loading).toBe(false);
    expect(data.state.error).toBeNull();
  });

  it('[LOCALIZATION-FE-S005] updateLocale: 失敗時に false を返し state.error が設定される', async () => {
    const { data, actions } = useAccount();

    actions.applySnapshot('account-123', 'ja');

    vi.spyOn(accountApi, 'updateSettings').mockRejectedValue(new Error('network error'));

    const result = await actions.updateLocale('en', { Authorization: 'Bearer test' });

    expect(result).toBe(false);
    expect(data.state.loading).toBe(false);
    expect(data.state.error).toBe('account-settings-update-failed');
  });

  it('[LOCALIZATION-FE-S004] applySnapshot: accountId と locale を state に反映する', () => {
    const { data, actions } = useAccount();

    actions.applySnapshot('account-123', 'en');

    expect(data.state.account).not.toBeNull();
    expect(data.state.account?.id).toBe('account-123');
    expect(data.state.account?.setting.locale).toBe('en');
  });
});

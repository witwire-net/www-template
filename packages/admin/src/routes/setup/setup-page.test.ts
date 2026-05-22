import { beforeEach, describe, expect, it, vi } from 'vitest';

const setupPageMocks = vi.hoisted(() => ({
  countOperators: vi.fn(),
  getAdminPrisma: vi.fn(),
  getAdminBootstrapConfig: vi.fn(),
}));

vi.mock('$lib/server/models/operators.js', () => ({
  countOperators: setupPageMocks.countOperators,
}));
vi.mock('$lib/server/infrastructure/db/prisma.js', () => ({
  getAdminPrisma: setupPageMocks.getAdminPrisma,
}));
vi.mock('$lib/server/infrastructure/config/env.js', () => ({
  getAdminBootstrapConfig: setupPageMocks.getAdminBootstrapConfig,
}));

import { load } from './+page.server.js';

describe('initial setup page server contract', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('operator が存在する場合は初回 setup フォームを表示せず login へ戻す', async () => {
    setupPageMocks.countOperators.mockResolvedValue(1);
    setupPageMocks.getAdminBootstrapConfig.mockReturnValue(enabledBootstrapGate());

    await expect(load({} as never)).rejects.toMatchObject({ status: 303, location: '/login' });
  });

  it('bootstrap gate が無効または期限切れの場合は初回 setup フォームを表示しない', async () => {
    setupPageMocks.countOperators.mockResolvedValue(0);
    setupPageMocks.getAdminBootstrapConfig.mockReturnValueOnce({
      adminBootstrapEnabled: false,
      adminBootstrapExpiresAt: new Date('2999-01-01T00:00:00.000Z'),
    });
    await expect(load({} as never)).rejects.toMatchObject({ status: 403 });
    expect(setupPageMocks.countOperators).not.toHaveBeenCalled();

    setupPageMocks.getAdminBootstrapConfig.mockReturnValueOnce({
      adminBootstrapEnabled: true,
      adminBootstrapExpiresAt: new Date(0),
    });
    await expect(load({} as never)).rejects.toMatchObject({ status: 403 });
    expect(setupPageMocks.countOperators).not.toHaveBeenCalled();
  });
});

function enabledBootstrapGate(): { adminBootstrapEnabled: boolean; adminBootstrapExpiresAt: Date } {
  return {
    adminBootstrapEnabled: true,
    adminBootstrapExpiresAt: new Date('2999-01-01T00:00:00.000Z'),
  };
}

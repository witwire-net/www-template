import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import ProfilesConsole from './ProfilesConsole.svelte';

describe('ProfilesConsole', () => {
  it('renders app api guidance', async () => {
    render(ProfilesConsole);

    expect(await screen.findByText('プロフィール画面プレースホルダー')).toBeInTheDocument();
    expect(screen.getByText(/専用の `\/api\/v1\/app\/\*` endpoint が有効/)).toBeInTheDocument();
  });

  it('explains bearer token auth boundary', async () => {
    render(ProfilesConsole);

    await waitFor(() => {
      expect(screen.getByText(/Authorization: Bearer <token>/)).toBeInTheDocument();
    });
  });
});

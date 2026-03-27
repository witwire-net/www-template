import Collapsible from './Collapsible.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Disclosure/Collapsible',
  component: Collapsible,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof Collapsible>;

export default meta;

type Story = StoryObj<typeof meta>;

/** デフォルト：折りたたみ状態で開始 */
export const Default: Story = {
  args: {
    trigger: 'セクション 1',
    content: 'これは折りたたみコンテンツです。トリガーをクリックすると展開・折りたたみができます。',
  },
};

/** 初期展開状態 */
export const DefaultOpen: Story = {
  args: {
    trigger: '最初から展開済み',
    content: 'defaultOpen={true} で初期状態を展開にできます。',
    defaultOpen: true,
  },
};

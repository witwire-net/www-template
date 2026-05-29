import { createCustomerAccount, getAdminAccountDetail, searchAdminAccounts } from '../accounts';

import type {
  AdminAccountDomainError,
  AdminAccountListItem,
  AdminAccountCreateInput,
} from '../accounts';

interface AdminAccountsState {
  query: string;
  status: string;
  cursor: string | null;
  nextCursor: string | null;
  currentPage: number;
  accounts: AdminAccountListItem[];
  isLoading: boolean;
  isCreating: boolean;
  listError: AdminAccountDomainError | null;
  createError: AdminAccountDomainError | null;
  createEmail: string;
  createLocale: NonNullable<AdminAccountCreateInput['locale']>;
}

interface AdminAccountDetailState {
  account: AdminAccountListItem | null;
  isLoading: boolean;
  detailError: AdminAccountDomainError | null;
  loadedAccountId: string | null;
}

interface AdminAccountsData {
  state: AdminAccountsState;
}

interface AdminAccountDetailData {
  state: AdminAccountDetailState;
}

interface AdminAccountsActions {
  loadFromSearchParams: (params: URLSearchParams) => Promise<void>;
  applyFilters: () => void;
  changePage: (pageNumber: number) => void;
  submitCreateAccount: () => Promise<void>;
  openAccount: (id: string) => void;
}

interface AdminAccountDetailActions {
  loadAccountDetail: (accountId: string) => Promise<void>;
}

interface AdminAccountsOptions {
  readSearchParams: () => URLSearchParams;
  navigateTo: (url: string) => void;
}

interface AdminAccountDetailOptions {
  readAccountId: () => string;
}

function createInitialAccountsState(): AdminAccountsState {
  // 一覧検索・作成 form・cursor pagination を同じ state に集め、URL search を唯一の検索 source にする。
  return {
    query: '',
    status: '',
    cursor: null,
    nextCursor: null,
    currentPage: 1,
    accounts: [],
    isLoading: false,
    isCreating: false,
    listError: null,
    createError: null,
    createEmail: '',
    createLocale: 'ja',
  };
}

function createInitialAccountDetailState(): AdminAccountDetailState {
  // detail route は accountId ごとに読み込み状態を分離し、古い detail error を残さない。
  return { account: null, isLoading: false, detailError: null, loadedAccountId: null };
}

function buildAccountsUrl(
  state: AdminAccountsState,
  pageNumber: number,
  nextPageCursor: string | null = null
): string {
  // 画面状態を URL に正規化し、再読み込みや共有でも同じ検索条件を復元できるようにする。
  const params: string[] = [];
  if (state.query !== '') params.push(`query=${encodeURIComponent(state.query)}`);
  if (state.status !== '') params.push(`status=${encodeURIComponent(state.status)}`);
  if (nextPageCursor !== null) params.push(`cursor=${encodeURIComponent(nextPageCursor)}`);
  params.push(`page=${encodeURIComponent(String(pageNumber))}`);
  return `/accounts?${params.join('&')}`;
}

/**
 * Admin account 一覧と作成 form を扱う domain composable です。
 *
 * route component は URL search の読み取りと navigation callback だけを提供し、
 * account list/detail の I/O は Admin domain action に集約します。
 */
function useAdminAccounts(options: AdminAccountsOptions): {
  data: AdminAccountsData;
  actions: AdminAccountsActions;
} {
  const state = $state<AdminAccountsState>(createInitialAccountsState());

  const actions: AdminAccountsActions = {
    loadFromSearchParams: async (params) => {
      // SvelteKit server load を使わず、domain function 経由で Admin API の account list を取得する。
      state.query = params.get('query') ?? '';
      state.status = params.get('status') ?? '';
      state.cursor = params.get('cursor');
      state.currentPage = Number(params.get('page') ?? '1');
      state.isLoading = true;
      state.listError = null;

      try {
        // backend contract は email/cursor/limit を受けるため、status は UI 表示条件として適用する。
        const result = await searchAdminAccounts({
          email: state.query,
          cursor: state.cursor ?? undefined,
          limit: 20,
        });
        if (!result.success) {
          state.accounts = [];
          state.listError = result.error;
          return;
        }

        // status filter が URL にある場合だけ表示を絞り、Account lifecycle の source of truth は backend response に保つ。
        state.accounts =
          state.status === ''
            ? result.data.accounts
            : result.data.accounts.filter((account) => account.status === state.status);
        state.nextCursor = result.data.nextCursor;
      } finally {
        // 成功・失敗に関わらず loading を解除し、再検索できる状態へ戻す。
        state.isLoading = false;
      }
    },
    applyFilters: () => {
      // filter 操作は URL 更新に寄せ、一覧取得は URL 監視 effect から一貫して実行する。
      options.navigateTo(buildAccountsUrl(state, 1));
    },
    changePage: (pageNumber) => {
      // cursor pagination なので、次ページは backend が返した opaque cursor がある場合だけ進める。
      if (pageNumber > state.currentPage && state.nextCursor !== null) {
        options.navigateTo(buildAccountsUrl(state, pageNumber, state.nextCursor));
        return;
      }

      // 前ページは cursor history を保持しないため、先頭ページへ戻して過去 cursor の誤用を避ける。
      options.navigateTo(buildAccountsUrl(state, 1));
    },
    submitCreateAccount: async () => {
      // Account 作成は page から Admin API wrapper を直接 import せず、domain function へ委譲する。
      if (state.isCreating) return;
      state.isCreating = true;
      state.createError = null;

      try {
        const result = await createCustomerAccount({
          email: state.createEmail,
          locale: state.createLocale,
        });
        if (!result.success) {
          state.createError = result.error;
          return;
        }

        // 作成成功後は入力を消し、作成済み account の詳細へ遷移する。
        state.createEmail = '';
        options.navigateTo(`/accounts/${result.data.id}`);
      } finally {
        // backend validation / network failure のどちらでも form を再操作可能にする。
        state.isCreating = false;
      }
    },
    openAccount: (id) => {
      // 一覧行選択は accountId を URL に載せるだけにし、detail 読み込みは detail composable に分離する。
      options.navigateTo(`/accounts/${id}`);
    },
  };

  $effect(() => {
    // URL search の変化を domain composable が監視し、route component から $effect I/O を排除する。
    void actions.loadFromSearchParams(options.readSearchParams());
  });

  return { data: { state }, actions };
}

/**
 * Admin account detail を route param に同期して読み込む domain composable です。
 */
function useAdminAccountDetail(options: AdminAccountDetailOptions): {
  data: AdminAccountDetailData;
  actions: AdminAccountDetailActions;
} {
  const state = $state<AdminAccountDetailState>(createInitialAccountDetailState());

  const actions: AdminAccountDetailActions = {
    loadAccountDetail: async (accountId) => {
      // accountId が変わるたびに前回 error を消し、現在 route の結果だけを表示する。
      state.account = null;
      state.detailError = null;
      state.isLoading = true;
      state.loadedAccountId = accountId;

      try {
        const result = await getAdminAccountDetail(accountId);
        if (state.loadedAccountId !== accountId) return;
        if (!result.success) {
          state.account = null;
          state.detailError = result.error;
          return;
        }

        // Admin API read model を detail 表示用 state に反映し、passkey 数も metadata として表示する。
        state.account = result.data;
      } finally {
        if (state.loadedAccountId === accountId) state.isLoading = false;
      }
    },
  };

  $effect(() => {
    // SvelteKit route param 変化を domain composable が監視し、page-level $effect を不要にする。
    void actions.loadAccountDetail(options.readAccountId());
  });

  return { data: { state }, actions };
}

export { useAdminAccountDetail, useAdminAccounts };

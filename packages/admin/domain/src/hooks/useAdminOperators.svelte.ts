import { createAdminOperator } from '../operators';

import type { AdminOperatorDomainError, AdminOperatorCreateResult } from '../operators';

type AdminOperatorRoleInput = 'admin' | 'operator' | 'viewer';

interface AdminOperatorRow {
  id: string;
  email: string;
  displayName: string;
  role: string;
  isActive: boolean;
  lastLoginAt: Date | null;
}

interface AdminOperatorsState {
  createdOperators: AdminOperatorRow[];
  addOpen: boolean;
  newOperatorEmail: string;
  newOperatorRole: string;
  isCreating: boolean;
  createError: AdminOperatorDomainError | null;
}

interface AdminOperatorsData {
  state: AdminOperatorsState;
}

interface AdminOperatorsActions {
  submitCreateOperator: () => Promise<void>;
}

function createInitialOperatorsState(): AdminOperatorsState {
  // operator 作成 dialog の state を domain に集約し、page から API wrapper を直接呼ばせない。
  return {
    createdOperators: [],
    addOpen: false,
    newOperatorEmail: '',
    newOperatorRole: 'viewer',
    isCreating: false,
    createError: null,
  };
}

function toOperatorRole(role: string): AdminOperatorRoleInput {
  // Select から来る文字列を contract の role union に絞り、未知値は最小権限 viewer に落とす。
  if (role === 'admin' || role === 'operator') return role;
  return 'viewer';
}

function toOperatorRow(result: AdminOperatorCreateResult): AdminOperatorRow {
  // response に setup token 平文は含まれないため、作成済み summary だけを一覧に即時反映する。
  return {
    id: result.id,
    email: result.email,
    displayName: result.email,
    role: result.role,
    isActive: result.active,
    lastLoginAt: null,
  };
}

/**
 * Admin operators 管理画面の作成 dialog state と mutation を扱う domain composable です。
 */
function useAdminOperators(): { data: AdminOperatorsData; actions: AdminOperatorsActions } {
  const state = $state<AdminOperatorsState>(createInitialOperatorsState());

  const actions: AdminOperatorsActions = {
    submitCreateOperator: async () => {
      // operator 作成は二重送信を止め、Admin API の Bearer/RBAC 検証へ一度だけ委譲する。
      if (state.isCreating) return;
      state.isCreating = true;
      state.createError = null;

      try {
        const result = await createAdminOperator({
          email: state.newOperatorEmail,
          role: toOperatorRole(state.newOperatorRole),
        });
        if (!result.success) {
          state.createError = result.error;
          return;
        }

        // 成功時だけ optimistic row を追加し、入力と dialog state を初期化する。
        state.createdOperators = [toOperatorRow(result.data), ...state.createdOperators];
        state.newOperatorEmail = '';
        state.newOperatorRole = 'viewer';
        state.addOpen = false;
      } finally {
        // 成功・失敗のどちらでも form を再操作可能に戻す。
        state.isCreating = false;
      }
    },
  };

  return { data: { state }, actions };
}

export { useAdminOperators };

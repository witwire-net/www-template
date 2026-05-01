import type { StatusState } from './types';

function createStatusInitialState(): StatusState {
  return {
    error: undefined,
    isLoading: false,
    message: '',
    timestamp: null,
  };
}

function toStatusErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return '公開ステータスの取得に失敗しました。';
}

export { createStatusInitialState, toStatusErrorMessage };

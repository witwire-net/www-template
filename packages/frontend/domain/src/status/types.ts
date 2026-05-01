interface StatusState {
  error?: string;
  isLoading: boolean;
  message: string;
  timestamp: Date | null;
}

export type { StatusState };

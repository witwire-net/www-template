import { describe, expect, it } from 'vitest';

import {
  applyRecoveryAccepted,
  createGenericRecoverySentView,
  createRecoveryFlowInitialState,
} from './state';

describe('recoveryState', () => {
  it('[AUTH-FE-S003] maps accepted recovery requests to a generic sent view', () => {
    const registeredState = createRecoveryFlowInitialState();
    const throttledState = createRecoveryFlowInitialState();

    applyRecoveryAccepted(registeredState, '01ARZ3NDEKTSV4RRFFQ69G5FAV', 'no-store');
    applyRecoveryAccepted(throttledState, '01ARZ3NDEKTSV4RRFFQ69G5FAW', 'no-store');

    expect(registeredState.phase).toBe('sent');
    expect(throttledState.phase).toBe('sent');
    expect(registeredState.sentView).toEqual(createGenericRecoverySentView());
    expect(throttledState.sentView).toEqual(createGenericRecoverySentView());
    expect(registeredState.sentView.description).not.toContain('アカウント');
  });
});

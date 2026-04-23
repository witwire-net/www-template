import 'vitest';
import type { TestingLibraryMatchers } from '@testing-library/jest-dom/matchers';

declare module 'vitest' {
  interface Assertion<T = unknown> extends TestingLibraryMatchers<unknown, T> {
    _jestDomAssertion?: never;
  }
  interface AsymmetricMatchersContaining extends TestingLibraryMatchers<unknown, unknown> {
    _jestDomAsymmetric?: never;
  }
}

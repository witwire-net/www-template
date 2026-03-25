import type { VirtualItem } from '@tanstack/svelte-virtual';

/** Allowed key types for virtual items (matches `@tanstack/virtual-core` Key). */
export type VirtualItemKey = number | string | bigint;

/**
 * Configuration for opt-in list virtualization.
 *
 * When provided, the component renders only visible items
 * using `@tanstack/svelte-virtual` with dynamic measurement.
 */
export interface VirtualizeOptions {
  /** Estimated height (px) of a single item. Used for initial layout before measurement. */
  estimateSize: number;
  /** Number of extra items to render outside the viewport. @default 5 */
  overscan?: number;
  /** Fixed height (px) for the scrollable container. @default 400 */
  height?: number;
  /** Gap (px) between items, matching the non-virtualized layout. @default 0 */
  gap?: number;
  /** Returns a stable key for the item at the given index. Used by the virtualizer for measurement caching and Svelte DOM reconciliation. */
  getItemKey?: (index: number) => VirtualItemKey;
}

/** Re-export VirtualItem for consumer convenience. */
export type { VirtualItem };

/** Default overscan used when the caller omits it. */
export const DEFAULT_OVERSCAN = 5;

/** Default container height used when the caller omits it. */
export const DEFAULT_HEIGHT = 400;

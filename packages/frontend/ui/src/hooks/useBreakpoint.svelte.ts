/**
 * useBreakpoint — reactive device-class detection hook (Svelte 5 Runes)
 *
 * ## Breakpoint system
 *
 * ### Device tiers (used by useBreakpoint())
 *   mobile  : ≤ 767px
 *   tablet  : 768px – 1023px
 *   desktop : ≥ 1024px
 *
 * ### Grid tiers (Bootstrap-compatible, used by Grid/Container)
 *   xs  : 0px      (no min-width constraint)
 *   sm  : ≥ 576px
 *   md  : ≥ 768px
 *   lg  : ≥ 992px
 *   xl  : ≥ 1200px
 *   2xl : ≥ 1400px
 *
 * ### Content fold points (component-level wrapping)
 *   XS_FOLD  : 360px  — very small modals / compact widgets
 *   SM_FOLD  : 520px  — small modal max-width
 *   MD_FOLD  : 560px  — collection: 1-col threshold
 *   LG_FOLD  : 640px  — inline-form stack threshold
 *   XL_FOLD  : 720px  — 2-col threshold (invoices, webhooks, collection)
 *   2XL_FOLD : 860px  — hero section mid breakpoint
 *   3XL_FOLD : 960px  — large modal / collection 3-col threshold
 *
 * Usage inside a Svelte component or another .svelte.ts file:
 *
 *   import { useBreakpoint, BREAKPOINTS } from '@ui/hooks/useBreakpoint.svelte';
 *
 *   const bp = useBreakpoint();
 *   // bp.isMobile, bp.isTablet, bp.isDesktop, bp.type
 */

/** All breakpoint boundaries (px) — single source of truth for JS/TS and SCSS. */
export const BREAKPOINTS = {
  // ── Device tiers ────────────────────────────────────────────────────────────
  /** Upper bound of the mobile range (px, inclusive). CSS: max-width: 767px */
  MOBILE_MAX: 767,
  /** Lower bound of the tablet range (px, inclusive). CSS: min-width: 768px */
  TABLET_MIN: 768,
  /** Upper bound of the tablet range (px, inclusive). CSS: max-width: 1023px */
  TABLET_MAX: 1023,
  /** Lower bound of the desktop range (px, inclusive). CSS: min-width: 1024px */
  DESKTOP_MIN: 1024,

  // ── Grid tiers (Bootstrap-compatible) ───────────────────────────────────────
  /** Grid sm breakpoint — min-width: 576px */
  GRID_SM: 576,
  /** Grid md breakpoint — min-width: 768px */
  GRID_MD: 768,
  /** Grid lg breakpoint — min-width: 992px */
  GRID_LG: 992,
  /** Grid xl breakpoint — min-width: 1200px */
  GRID_XL: 1200,
  /** Grid 2xl breakpoint — min-width: 1400px */
  GRID_2XL: 1400,

  // ── Content fold points (component-level wrapping thresholds) ────────────────
  /** Very small modal / compact widget max-width (360px) */
  FOLD_XS: 360,
  /** Small modal max-width (520px) */
  FOLD_SM: 520,
  /** Collection 1-col / story-showcase stack threshold (560px) */
  FOLD_MD: 560,
  /** Inline-form stack threshold (640px) */
  FOLD_LG: 640,
  /** 2-col threshold: invoices, webhooks, collection (720px) */
  FOLD_XL: 720,
  /** Hero section mid breakpoint (860px) */
  FOLD_2XL: 860,
  /** Large modal / collection 3-col threshold (960px) */
  FOLD_3XL: 960,
} as const;

/** Current device classification derived from viewport width. */
export type DeviceType = 'desktop' | 'mobile' | 'tablet';

/** Return type of {@link useBreakpoint}. */
export type UseBreakpointReturn = {
  /** Current device classification */
  readonly type: DeviceType;
  readonly isMobile: boolean;
  readonly isTablet: boolean;
  readonly isDesktop: boolean;
};

/**
 * Returns a reactive object that reflects the current viewport device class.
 * Safe to call during SSR — defaults to 'desktop' until the browser reports otherwise.
 *
 * Must be called at component initialisation time (i.e. not inside callbacks).
 */
export function useBreakpoint(): UseBreakpointReturn {
  let type = $state<DeviceType>(resolveType());

  $effect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const mobileQuery = window.matchMedia(`(max-width: ${String(BREAKPOINTS.MOBILE_MAX)}px)`);
    const tabletQuery = window.matchMedia(
      `(min-width: ${String(BREAKPOINTS.TABLET_MIN)}px) and (max-width: ${String(BREAKPOINTS.TABLET_MAX)}px)`
    );

    const update = (): void => {
      if (mobileQuery.matches) {
        type = 'mobile';
      } else if (tabletQuery.matches) {
        type = 'tablet';
      } else {
        type = 'desktop';
      }
    };

    update();
    mobileQuery.addEventListener('change', update);
    tabletQuery.addEventListener('change', update);

    return () => {
      mobileQuery.removeEventListener('change', update);
      tabletQuery.removeEventListener('change', update);
    };
  });

  return {
    get type() {
      return type;
    },
    get isMobile() {
      return type === 'mobile';
    },
    get isTablet() {
      return type === 'tablet';
    },
    get isDesktop() {
      return type === 'desktop';
    },
  };
}

/** Synchronous initial resolution (runs once before effects). */
function resolveType(): DeviceType {
  if (typeof window === 'undefined') {
    return 'desktop';
  }
  if (window.matchMedia(`(max-width: ${String(BREAKPOINTS.MOBILE_MAX)}px)`).matches) {
    return 'mobile';
  }
  if (
    window.matchMedia(
      `(min-width: ${String(BREAKPOINTS.TABLET_MIN)}px) and (max-width: ${String(BREAKPOINTS.TABLET_MAX)}px)`
    ).matches
  ) {
    return 'tablet';
  }
  return 'desktop';
}

const palette = {
  brand: {
    400: '#4ac6c8',
    500: '#5fd9db',
    600: '#3daeb0',
  },
  neutral: {
    50: '#fafafa',
    100: '#f4f4f4',
    200: '#e6e6e6',
    300: '#d4d4d4',
    400: '#a3a3a3',
    500: '#737373',
    600: '#575757',
    700: '#424242',
    800: '#333333',
    900: '#292929',
    950: '#1f1f1f',
  },
  status: {
    error: '#ff7675',
    info: '#74b9ff',
    success: '#00b894',
    warning: '#fdcb6e',
  },
  white: '#ffffff',
} as const;

const colors = {
  background: palette.neutral[50],
  border: palette.neutral[200],
  borderHover: palette.brand[500],
  borderSubtle: palette.neutral[100],
  primary: palette.brand[500],
  primaryActive: palette.brand[600],
  primaryContrast: palette.neutral[950],
  primaryHover: palette.brand[400],
  surface: palette.white,
  surfaceFloating: palette.white,
  surfaceHover: palette.neutral[100],
  text: palette.neutral[900],
  textMuted: palette.neutral[400],
  textOnBrand: palette.neutral[950],
  textSecondary: palette.neutral[600],
} as const;

const spacing = {
  xs: '0.25rem',
  sm: '0.5rem',
  md: '1rem',
  lg: '1.5rem',
  xl: '2rem',
  x2l: '3rem',
  x3l: '5rem',
} as const;

const typography = {
  families: {
    display: "'M PLUS 1', system-ui, -apple-system, sans-serif",
    mono: "'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    sans: "'Noto Sans JP', system-ui, -apple-system, sans-serif",
  },
  lineHeights: {
    normal: 1.5,
    relaxed: 1.65,
    tight: 1.2,
  },
  sizes: {
    x2s: '0.8125rem',
    xs: '0.75rem',
    sm: '0.875rem',
    base: '1rem',
    lg: '1.125rem',
    xl: '1.25rem',
    x2l: '1.5rem',
    x3l: '2.5rem',
  },
  weights: {
    regular: 400,
    medium: 500,
    bold: 700,
    extraBold: 800,
    black: 900,
  },
} as const;

const radius = {
  sm: '0.5rem',
  md: '1rem',
  lg: '1.5rem',
  xl: '2rem',
  full: '9999px',
} as const;

const shadow = {
  glow: 'none',
  lg: 'none',
  md: 'none',
  sm: 'none',
} as const;

const transitions = {
  bounce: '500ms cubic-bezier(0.34, 1.56, 0.64, 1)',
  fast: '150ms ease-out',
  normal: '300ms cubic-bezier(0.4, 0, 0.2, 1)',
  smooth: '320ms cubic-bezier(0.22, 1, 0.36, 1)',
} as const;

/** Svelte 移行で共有する UI トークン契約。 */
export const theme = {
  colors,
  dataAttribute: 'data-theme',
  defaultMode: 'light',
  modes: {
    dark: {
      background: palette.neutral[950],
      border: palette.neutral[800],
      borderHover: palette.brand[500],
      borderSubtle: palette.neutral[900],
      surface: palette.neutral[900],
      surfaceFloating: palette.neutral[900],
      surfaceHover: palette.neutral[800],
      text: palette.neutral[50],
      textMuted: palette.neutral[400],
      textSecondary: palette.neutral[300],
    },
    light: colors,
  },
  palette,
  radius,
  shadow,
  spacing,
  typography,
  transitions,
} as const;

/** UI トークン契約の型。 */
export type Theme = typeof theme;

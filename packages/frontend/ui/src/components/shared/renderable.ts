import type { Snippet } from 'svelte';

/**
 * Renderable values accepted by shared Svelte UI helpers.
 */
export type Renderable = Snippet | string | number | null | undefined;

/**
 * Joins class names while dropping empty values.
 */
export function joinClassNames(...classNames: (string | false | null | undefined)[]): string {
  return classNames
    .filter((value): value is string => typeof value === 'string' && value !== '')
    .join(' ');
}

/**
 * Narrows a renderable value to a Svelte snippet.
 */
export function isSnippet(value: Renderable): value is Snippet {
  return typeof value === 'function';
}

/**
 * Converts plain renderable values to text output.
 */
export function getTextContent(value: Renderable): string {
  if (typeof value === 'string') {
    return value;
  }

  if (typeof value === 'number') {
    return String(value);
  }

  return '';
}

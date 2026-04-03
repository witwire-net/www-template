import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

/** Tailwind class name をマージする。 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/** `child` prop を除いた props 型。 */
export type WithoutChild<T> = T extends { child?: unknown } ? Omit<T, 'child'> : T;
/** `children` prop を除いた props 型。 */
export type WithoutChildren<T> = T extends { children?: unknown } ? Omit<T, 'children'> : T;
/** `child` と `children` の両方を除いた props 型。 */
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
/** `ref` prop を付与した props 型。 */
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };

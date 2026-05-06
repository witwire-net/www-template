import { getContext, hasContext, setContext } from 'svelte';

import type { WithElementRef } from '@ui/lib/utils';

import type {
  EmblaCarouselSvelteType,
  default as emblaCarouselSvelte,
} from 'embla-carousel-svelte';
import type { HTMLAttributes } from 'svelte/elements';

/** Embla carousel API 型。 */
export type CarouselAPI =
  NonNullable<NonNullable<EmblaCarouselSvelteType['$$_attributes']>['on:emblaInit']> extends (
    evt: CustomEvent<infer CarouselAPI>
  ) => void
    ? CarouselAPI
    : never;

type EmblaCarouselConfig = NonNullable<Parameters<typeof emblaCarouselSvelte>[1]>;

/** Carousel options 型。 */
export type CarouselOptions = EmblaCarouselConfig['options'];
/** Carousel plugin 配列型。 */
export type CarouselPlugins = EmblaCarouselConfig['plugins'];

////

/** Carousel root props 型。 */
export type CarouselProps = {
  opts?: CarouselOptions;
  plugins?: CarouselPlugins;
  setApi?: (api: CarouselAPI | undefined) => void;
  orientation?: 'horizontal' | 'vertical';
} & WithElementRef<HTMLAttributes<HTMLDivElement>>;

const EMBLA_CAROUSEL_CONTEXT = Symbol('EMBLA_CAROUSEL_CONTEXT');

/** Carousel context で共有する状態。 */
export interface EmblaContext {
  api: CarouselAPI | undefined;
  orientation: 'horizontal' | 'vertical';
  scrollNext: () => void;
  scrollPrev: () => void;
  canScrollNext: boolean;
  canScrollPrev: boolean;
  handleKeyDown: (e: KeyboardEvent) => void;
  options: CarouselOptions;
  plugins: CarouselPlugins;
  onInit: (e: CustomEvent<CarouselAPI>) => void;
  scrollTo: (index: number, jump?: boolean) => void;
  scrollSnaps: number[];
  selectedIndex: number;
}

/** Carousel context を設定する。 */
export function setEmblaContext(config: EmblaContext): EmblaContext {
  setContext(EMBLA_CAROUSEL_CONTEXT, config);
  return config;
}

/** Carousel context を取得する。 */
export function getEmblaContext(name = 'This component'): EmblaContext {
  if (!hasContext(EMBLA_CAROUSEL_CONTEXT)) {
    throw new Error(`${name} must be used within a <Carousel.Root> component`);
  }
  return getContext<ReturnType<typeof setEmblaContext>>(EMBLA_CAROUSEL_CONTEXT);
}

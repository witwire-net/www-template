import { getContext, setContext, type Component, type Snippet } from 'svelte';

import type { Tooltip } from 'layerchart';

type ChartConfigEntry = {
  label?: string;
  icon?: Component;
} & (
  | { color?: string; theme?: never }
  | { color?: never; theme: Record<keyof typeof THEMES, string> }
);

/** ライト/ダークの chart selector 定義。 */
export const THEMES = { light: '', dark: '.dark' } as const;

/** Chart component が参照する series config。 */
export type ChartConfig = Record<string, ChartConfigEntry>;

/** Snippet の第 1 引数型を取り出す。 */
export type ExtractSnippetParams<T> = T extends Snippet<[infer P]> ? P : never;

/** layerchart tooltip payload 型。 */
export type TooltipPayload = Tooltip.TooltipSeries;

function getStringRecordValue(source: object | null | undefined, key: string): string | undefined {
  if (source == null || !Object.hasOwn(source, key)) {
    return undefined;
  }

  const value = Reflect.get(source, key) as unknown;

  return typeof value === 'string' ? value : undefined;
}

/** Tooltip payload から series config を引く。 */
export function getPayloadConfigFromPayload(
  config: ChartConfig,
  payload: TooltipPayload,
  key: string,
  data?: Record<string, unknown> | null
): ChartConfigEntry | undefined {
  const payloadConfig = payload.config as object;

  let configLabelKey: string = key;
  const payloadValue = getStringRecordValue(payload, key);
  const payloadConfigValue = getStringRecordValue(payloadConfig, key);
  const dataValue = getStringRecordValue(data, key);

  if (payload.key === key) {
    configLabelKey = payload.key;
  } else if (payload.label === key) {
    configLabelKey = payload.label;
  } else if (payloadValue !== undefined) {
    configLabelKey = payloadValue;
  } else if (payloadConfigValue !== undefined) {
    configLabelKey = payloadConfigValue;
  } else if (dataValue !== undefined) {
    configLabelKey = dataValue;
  }

  const preferredConfig = Reflect.get(config, configLabelKey) as ChartConfigEntry | undefined;

  if (preferredConfig !== undefined) {
    return preferredConfig;
  }

  return Reflect.get(config, key) as ChartConfigEntry | undefined;
}

interface ChartContextValue {
  config: ChartConfig;
}

const chartContextKey = Symbol('chart-context');

/** Chart context を設定する。 */
export function setChartContext(value: ChartContextValue) {
  return setContext(chartContextKey, value);
}

/** Chart context を取得する。 */
export function useChart() {
  return getContext<ChartContextValue>(chartContextKey);
}

declare module '*.module.scss' {
  const classNames: Record<string, string>;
  /**
   * class classNames.
   */
  export default classNames;
}

declare module '*.scss';

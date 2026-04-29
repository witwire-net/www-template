export default {
  extends: ['stylelint-config-standard'],
  rules: {
    'at-rule-disallowed-list': ['apply'],
    'at-rule-empty-line-before': null,
    'at-rule-no-unknown': [
      true,
      {
        ignoreAtRules: ['theme', 'custom-variant', 'layer', 'import'],
      },
    ],
    'block-no-empty': null,
    'color-hex-length': null,
    'custom-property-empty-line-before': null,
    'declaration-property-value-no-unknown': true,
    'hue-degree-notation': null,
    'import-notation': null,
    'lightness-notation': null,
    'no-duplicate-selectors': true,
    'selector-class-pattern': null,
    'value-keyword-case': null,
  },
};

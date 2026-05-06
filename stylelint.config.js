export default {
  extends: ['stylelint-config-standard'],
  rules: {
    'at-rule-disallowed-list': ['apply'],
    'at-rule-empty-line-before': null,
    'at-rule-no-unknown': [
      true,
      {
        ignoreAtRules: ['theme', 'custom-variant', 'layer', 'import', 'source'],
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
    'no-invalid-position-at-import-rule': null,
    'selector-class-pattern': null,
    'value-keyword-case': null,
  },
};

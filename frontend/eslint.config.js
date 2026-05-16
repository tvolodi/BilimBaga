import js from '@eslint/js'
import i18next from 'eslint-plugin-i18next'
import jsxA11y from 'eslint-plugin-jsx-a11y'

export default [
  js.configs.recommended,
  {
    files: ['src/**/*.{ts,tsx}'],
    plugins: { i18next, 'jsx-a11y': jsxA11y },
    rules: {
      'i18next/no-literal-string': ['warn', { markupOnly: true }],
      ...jsxA11y.configs.recommended.rules,
    },
  },
]

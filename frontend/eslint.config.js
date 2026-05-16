import js from '@eslint/js'
import i18next from 'eslint-plugin-i18next'

export default [
  js.configs.recommended,
  {
    files: ['src/**/*.{ts,tsx}'],
    plugins: { i18next },
    rules: {
      'i18next/no-literal-string': ['warn', { markupOnly: true }],
    },
  },
]

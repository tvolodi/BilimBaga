import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import i18next from 'eslint-plugin-i18next'
import jsxA11y from 'eslint-plugin-jsx-a11y'
import reactHooks from 'eslint-plugin-react-hooks'

export default [
  { ignores: ['dist/**', 'coverage/**', 'node_modules/**'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      globals: { ...globals.browser, ...globals.node },
    },
    plugins: { i18next, 'jsx-a11y': jsxA11y, 'react-hooks': reactHooks },
    rules: {
      'i18next/no-literal-string': ['warn', { markupOnly: true }],
      ...jsxA11y.configs.recommended.rules,
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      // Inputs here are custom components (non-DOM); autofocus in modals is intentional.
      'jsx-a11y/no-autofocus': ['error', { ignoreNonDOM: true }],
      // Labels often wrap controls with nested markup deeper than the default depth.
      'jsx-a11y/label-has-associated-control': ['error', { depth: 6 }],
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', destructuredArrayIgnorePattern: '^_' },
      ],
    },
  },
  {
    // shadcn primitives receive their content/association via props at the call site.
    files: ['src/components/ui/**/*.tsx'],
    rules: {
      'jsx-a11y/heading-has-content': 'off',
      'jsx-a11y/label-has-associated-control': 'off',
    },
  },
  {
    // Test fixtures are not user-facing copy.
    files: ['src/**/*.test.{ts,tsx}', 'src/test/**/*.{ts,tsx}'],
    rules: { 'i18next/no-literal-string': 'off' },
  },
]

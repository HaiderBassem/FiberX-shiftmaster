import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      // A leading underscore is this codebase's existing convention for a binding
      // that is deliberately unused (kept for API compatibility, or destructured
      // past). Honour it rather than forcing the convention to be abandoned.
      '@typescript-eslint/no-unused-vars': ['error', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
      }],

      // `any` is a warning rather than an error, so the existing occurrences do
      // not block every change while they are worked through. It is not
      // toothless: `npm run lint` caps warnings at the current count, so a new
      // `any` fails CI. Lower the cap as the number comes down.
      '@typescript-eslint/no-explicit-any': 'warn',

      // React Compiler diagnostics and fast-refresh ergonomics. These flag
      // missed optimisations and development-server niceties, not defects, and
      // clearing them means restructuring several large components — work worth
      // doing deliberately rather than as a side effect of a cleanup pass. They
      // are warnings so they stay visible and are covered by the same cap.
      'react-hooks/set-state-in-effect': 'warn',
      'react-hooks/preserve-manual-memoization': 'warn',
      'react-hooks/static-components': 'warn',
      'react-hooks/exhaustive-deps': 'warn',
      'react-refresh/only-export-components': 'warn',
    },
  },
])

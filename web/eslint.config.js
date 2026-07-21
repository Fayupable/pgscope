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
      globals: globals.browser,
    },
    rules: {
      // Downgraded from the plugin's default "error": this rule flags the
      // extremely common and correct "fetch data on mount" pattern (see
      // useInsights.ts) as a cascading-render risk. Worth a contributor's
      // attention, not worth blocking a correct PR over.
      'react-hooks/set-state-in-effect': 'warn',
    },
  },
  {
    // Architecture rule: every network call goes through shared/api/*.ts,
    // never called directly from a feature component or hook. This keeps
    // credentials handling, error shaping, and the base request pattern
    // in one place instead of duplicated per caller.
    files: ['src/features/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='fetch']",
          message: 'Do not call fetch() directly in a feature. Add or use a wrapper in shared/api/ instead.',
        },
        {
          selector: "NewExpression[callee.name='EventSource']",
          message: 'Do not construct EventSource directly in a feature. Use shared/api/sseClient.ts instead.',
        },
      ],
    },
  },
])
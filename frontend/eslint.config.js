import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'

export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'coverage'] },
  {
    files: ['**/*.{ts,tsx}'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs['recommended-latest'].rules,
      // Only-export-components is a fast-refresh perf hint, not a correctness
      // failure — keep it as a warning.
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
      // Render errors are surfaced via the ErrorBoundary; console.error there
      // is deliberate error reporting, so only ban debug-level logging.
      'no-console': ['error', { allow: ['warn', 'error'] }],
    },
  },
)

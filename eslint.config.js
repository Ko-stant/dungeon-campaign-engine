import js from '@eslint/js';
import globals from 'globals';
import prettier from 'eslint-config-prettier';
import tseslint from 'typescript-eslint';
import { defineConfig } from 'eslint/config';

const styleRules = {
  'prefer-const': 'error',
  'no-var': 'error',
  'eqeqeq': ['error', 'always'],
  'curly': ['error', 'all'],
  'quotes': ['error', 'single', { avoidEscape: true }],
  'object-curly-spacing': ['error', 'always'],
  'array-bracket-spacing': ['error', 'never'],
};

export default defineConfig(
  {
    ignores: [
      // Build output.
      'internal/web/static/dist/**',
      'build/**',
      'tmp/**',
    ],
  },
  js.configs.recommended,
  {
    // Root-level tooling config files run under Bun/Node.
    files: ['*.js', '*.mjs'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        ...globals.node,
      },
    },
    rules: {
      ...styleRules,
      'no-unused-vars': ['error', { vars: 'all', args: 'after-used' }],
    },
  },
  {
    // TypeScript client (internal/web/src) and build scripts.
    files: ['**/*.ts'],
    extends: [...tseslint.configs.strictTypeChecked, ...tseslint.configs.stylisticTypeChecked],
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
      globals: {
        ...globals.browser,
      },
    },
    rules: {
      ...styleRules,
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/restrict-template-expressions': ['error', { allowNumber: true }],
    },
  },
  {
    // Build scripts run under Bun, not the browser.
    files: ['scripts/**/*.ts'],
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
  },
  prettier,
);

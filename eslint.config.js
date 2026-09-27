import js from '@eslint/js';
import globals from 'globals';
import prettier from 'eslint-config-prettier';

const styleRules = {
  'no-unused-vars': ['error', {
    vars: 'all',
    args: 'after-used',
    ignoreRestSiblings: false,
  }],
  'prefer-const': 'error',
  'no-var': 'error',
  'eqeqeq': ['error', 'always'],
  'curly': ['error', 'all'],
  'quotes': ['error', 'single'],
  'object-curly-spacing': ['error', 'always'],
  'array-bracket-spacing': ['error', 'never'],
};

export default [
  {
    ignores: [
      // Legacy vanilla JS client: reference only, never converted, deleted in Phase 8.
      'internal/web/static/js/**',
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
    rules: styleRules,
  },
  prettier,
];

import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { buildFixtures, formatFixture } from './fixtures.ts';

const root = join(import.meta.dir, '..', '..');

describe('parity fixtures', () => {
  const fixtures = buildFixtures();
  for (const [path, data] of Object.entries(fixtures)) {
    test(`${path} is up to date (run bun run parity:gen)`, () => {
      expect(readFileSync(join(root, path), 'utf8')).toBe(formatFixture(data));
    });
  }

  test('the strike cases cover the rare outcomes', () => {
    const { cases } = fixtures['internal/combat/testdata/parity/hero_strikes.json'] as { cases: { result: { criticalMiss: boolean; crit: boolean; hit: boolean } }[] };
    expect(cases.some((c) => c.result.criticalMiss)).toBe(true);
    expect(cases.some((c) => c.result.crit)).toBe(true);
    expect(cases.some((c) => !c.result.hit && !c.result.criticalMiss)).toBe(true);
  });
});

/**
 * Writes the TS/Go parity fixtures (see scripts/parity/fixtures.ts):
 *   bun run parity:gen
 */

import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { buildFixtures, formatFixture } from './fixtures.ts';

const root = join(import.meta.dir, '..', '..');
for (const [path, data] of Object.entries(buildFixtures())) {
  const file = join(root, path);
  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, formatFixture(data));
  console.log(`wrote ${path}`);
}

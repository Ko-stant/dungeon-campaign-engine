/**
 * Fails when a tracked text file uses a British spelling (code, comments, UI
 * text and docs are all American English). Run by `make lint`.
 *
 *   bun scripts/check-spelling.ts
 */
import path from 'node:path';
import { findBritishSpellings } from './spelling.ts';

const root = path.resolve(import.meta.dir, '..');

/** Lock files. (The checker's own word lists are marked spelling:allow-file.) */
const SKIPPED = new Set(['bun.lock', 'go.sum']);

const listed = Bun.spawnSync(['git', 'ls-files', '-z'], { cwd: root });
if (listed.exitCode !== 0) {
  console.error(listed.stderr.toString());
  process.exit(1);
}
const files = listed.stdout
  .toString()
  .split('\0')
  .filter((file) => file !== '' && !SKIPPED.has(file));

let count = 0;
for (const file of files) {
  const bytes = await Bun.file(path.join(root, file)).bytes();
  if (bytes.includes(0)) {
    continue; // binary
  }
  for (const f of findBritishSpellings(new TextDecoder().decode(bytes))) {
    console.log(`${file}:${f.line}:${f.column}: "${f.word}" -> "${f.suggestion}"`);
    count++;
  }
}

if (count > 0) {
  console.error(`\n${count} British spelling(s); use American English (or mark the line spelling:allow).`);
  process.exit(1);
}

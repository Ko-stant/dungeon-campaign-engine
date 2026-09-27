/**
 * Bundles the TypeScript client. Every internal/web/src/pages/*.ts file is an
 * entrypoint and becomes internal/web/static/dist/<name>.js.
 *
 *   bun scripts/build.ts            one-off development build
 *   bun scripts/build.ts --minify   production build (make build)
 *   bun scripts/build.ts --watch    rebuild on change and ask the templ dev
 *                                   proxy (make dev) to reload the browser
 */
import { Glob } from 'bun';
import { watch } from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dir, '..');
const srcDir = path.join(root, 'internal/web/src');
const outDir = path.join(root, 'internal/web/static/dist');

const args = new Set(Bun.argv.slice(2));
const minify = args.has('--minify');
const watchMode = args.has('--watch');

function entrypoints(): string[] {
  const glob = new Glob('pages/*.ts');
  return [...glob.scanSync({ cwd: srcDir })]
    .filter((file) => !file.endsWith('.test.ts'))
    .map((file) => path.join(srcDir, file))
    .sort();
}

async function build(): Promise<boolean> {
  const started = performance.now();
  const result = await Bun.build({
    entrypoints: entrypoints(),
    outdir: outDir,
    target: 'browser',
    format: 'esm',
    sourcemap: 'linked',
    minify,
    naming: '[name].[ext]',
  });
  const ms = Math.round(performance.now() - started);
  if (!result.success) {
    for (const log of result.logs) {
      console.error(log);
    }
    console.error(`web build failed after ${ms}ms`);
    return false;
  }
  const files = result.outputs.filter((o) => o.kind === 'entry-point').map((o) => path.basename(o.path));
  console.log(`web build: ${files.join(', ') || '(no pages)'} in ${ms}ms`);
  return true;
}

/** Tells a running `templ generate --watch --proxy` to reload connected browsers. */
async function notifyTemplProxy(): Promise<void> {
  const proc = Bun.spawn(['go', 'tool', 'templ', 'generate', '--notify-proxy'], {
    cwd: root,
    stdout: 'ignore',
    stderr: 'ignore',
  });
  await proc.exited;
}

const ok = await build();
if (!watchMode) {
  process.exit(ok ? 0 : 1);
}

let timer: ReturnType<typeof setTimeout> | undefined;
watch(srcDir, { recursive: true }, (_event, filename) => {
  if (filename?.endsWith('.test.ts')) {
    return;
  }
  clearTimeout(timer);
  timer = setTimeout(() => {
    void build().then(async (built) => {
      if (built) {
        await notifyTemplProxy();
      }
    });
  }, 50);
});
console.log(`watching ${path.relative(root, srcDir)} for changes`);

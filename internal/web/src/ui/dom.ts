/** Minimal DOM construction helper for the TypeScript pages. */

type Child = Node | string | number | null | undefined | false;
type Attrs = Record<string, string | number | boolean | EventListener | null | undefined>;

/**
 * Creates an element. `class` sets className, `on<event>` keys add listeners,
 * booleans toggle attributes, and null/undefined values are skipped.
 */
export function h<K extends keyof HTMLElementTagNameMap>(tag: K, attrs: Attrs = {}, ...children: Child[]): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value === null || value === undefined || value === false) {
      continue;
    }
    if (key.startsWith('on') && typeof value === 'function') {
      el.addEventListener(key.slice(2).toLowerCase(), value);
    } else if (key === 'class') {
      el.className = String(value);
    } else if (value === true) {
      el.setAttribute(key, '');
    } else if (typeof value !== 'function') {
      el.setAttribute(key, String(value));
    }
  }
  append(el, children);
  return el;
}

export function append(parent: Node, children: readonly Child[]): void {
  for (const child of children) {
    if (child === null || child === undefined || child === false) {
      continue;
    }
    parent.appendChild(typeof child === 'string' || typeof child === 'number' ? document.createTextNode(String(child)) : child);
  }
}

/** Replaces all children of `parent`. */
export function replaceChildren(parent: Element, ...children: Child[]): void {
  parent.replaceChildren();
  append(parent, children);
}

/**
 * Remembers the focused text field under root (by aria-label or placeholder)
 * and returns a function that puts its value, caret and focus back on the
 * matching field after the panels are rebuilt, so a live update never wipes
 * what the GM is typing.
 */
export function preserveFocus(root: Element): () => void {
  const el = document.activeElement;
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) || !root.contains(el)) {
    return () => undefined;
  }
  const key = (e: Element): string | null => e.getAttribute('aria-label') ?? e.getAttribute('placeholder');
  const label = key(el);
  const { value, selectionStart, selectionEnd } = el;
  return () => {
    if (!label || el.isConnected) {
      return;
    }
    const next = [...root.querySelectorAll<HTMLInputElement | HTMLTextAreaElement>('input, textarea')].find((e) => key(e) === label);
    if (!next) {
      return;
    }
    const rendered = next.value;
    next.value = value;
    next.focus();
    if (value !== rendered) {
      // A programmatic value never fires "change" on blur; fire it once so a
      // restored edit is still saved, unless the browser already did.
      let changed = false;
      next.addEventListener('change', () => { changed = true; }, { once: true });
      next.addEventListener('blur', () => {
        if (!changed && next.value !== rendered) {
          next.dispatchEvent(new Event('change'));
        }
      }, { once: true });
    }
    try {
      next.setSelectionRange(selectionStart, selectionEnd);
    } catch {
      // number inputs have no caret position
    }
  };
}

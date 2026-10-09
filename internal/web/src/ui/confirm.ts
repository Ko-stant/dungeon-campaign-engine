/**
 * The Yes/No dialog that every delete asks through. Server-rendered forms opt
 * in with `data-confirm="<question>"` on the submit button (or the form) and
 * may relabel Yes with `data-confirm-yes`; the TypeScript pages call
 * confirmDialog directly. Only the Yes button goes ahead: No, Escape and a
 * click outside the dialog all keep the item.
 */
import { h } from './dom.ts';

/** Whether a closed dialog's return value means Yes. */
export function isYes(returnValue: string): boolean {
  return returnValue === 'yes';
}

/** The question to ask before a submit (the button's, else the form's); null submits without asking. */
export function confirmMessage(buttonText: string | undefined, formText: string | undefined): string | null {
  for (const text of [buttonText, formText]) {
    if (text?.trim()) {
      return text;
    }
  }
  return null;
}

/**
 * Remembers which forms the GM said Yes to: each approval lets exactly one
 * submit through, so answering No (approving nothing) keeps the form stopped.
 */
export class ConfirmGate<T extends object> {
  private readonly approved = new WeakSet<T>();

  approve(form: T): void {
    this.approved.add(form);
  }

  /** Whether this submit was approved; uses the approval up. */
  pass(form: T): boolean {
    return this.approved.delete(form);
  }
}

const dialogClass = 'm-auto w-full max-w-md rounded-lg border border-danger/60 bg-surface-2 p-5 text-content shadow-xl backdrop:bg-black/60';
const noClass = 'rounded-md border border-border/60 px-4 py-2 text-sm hover:border-amber-500';
const yesClass = 'rounded-md border border-danger/60 bg-danger/15 px-4 py-2 text-sm font-semibold text-danger hover:bg-danger/25';

/** Asks a Yes/No question in a modal dialog; resolves true only for Yes. No has the focus. */
export function confirmDialog(message: string, yesLabel = 'Yes, delete'): Promise<boolean> {
  const no = h('button', { type: 'submit', value: 'no', class: noClass, autofocus: true }, 'No');
  const dialog = h('dialog', { class: dialogClass, 'aria-label': 'Are you sure?' },
    h('form', { method: 'dialog', class: 'space-y-4' },
      h('h2', { class: 'text-lg font-semibold text-danger' }, 'Are you sure?'),
      h('p', { class: 'whitespace-pre-line text-sm' }, message),
      h('div', { class: 'flex justify-end gap-2' },
        no,
        h('button', { type: 'submit', value: 'yes', class: yesClass }, yesLabel))));
  // A click on the backdrop lands on the dialog itself: treat it as No.
  dialog.addEventListener('click', (ev) => {
    if (ev.target === dialog) {
      dialog.close('no');
    }
  });
  return new Promise((resolve) => {
    dialog.addEventListener('close', () => {
      dialog.remove();
      resolve(isYes(dialog.returnValue));
    }, { once: true });
    document.body.append(dialog);
    dialog.showModal();
    no.focus();
  });
}

/**
 * Makes every form submit whose button (or form) has data-confirm ask first.
 * The submit is stopped, the dialog opens, and only Yes submits it again with
 * the same button (so a button's formaction still applies).
 */
export function wireConfirmForms(root: Document): void {
  const gate = new ConfirmGate<HTMLFormElement>();
  root.addEventListener('submit', (ev) => {
    const form = ev.target;
    if (!(form instanceof HTMLFormElement)) {
      return;
    }
    const button = ev.submitter instanceof HTMLButtonElement || ev.submitter instanceof HTMLInputElement ? ev.submitter : null;
    const message = confirmMessage(button?.dataset.confirm, form.dataset.confirm);
    if (message === null || gate.pass(form)) {
      return;
    }
    ev.preventDefault();
    void confirmDialog(message, button?.dataset.confirmYes ?? form.dataset.confirmYes).then((yes) => {
      if (yes) {
        gate.approve(form);
        form.requestSubmit(button);
      }
    });
  }, true);
}

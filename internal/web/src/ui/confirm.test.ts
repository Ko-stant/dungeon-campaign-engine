import { describe, expect, test } from 'bun:test';
import { ConfirmGate, confirmMessage, isYes } from './confirm.ts';

describe('isYes', () => {
  test('only the Yes button confirms', () => {
    expect(isYes('yes')).toBe(true);
  });

  test('No, Escape, a click outside and anything else keep the item', () => {
    // No returns "no"; Escape and closing without a button leave "" (or "cancel").
    for (const value of ['no', '', 'cancel', 'Yes', 'yes ', 'true']) {
      expect(isYes(value)).toBe(false);
    }
  });
});

describe('confirmMessage', () => {
  test("the button's own text wins over the form's", () => {
    expect(confirmMessage('Delete the button thing?', 'Delete the form thing?')).toBe('Delete the button thing?');
    expect(confirmMessage(undefined, 'Delete the form thing?')).toBe('Delete the form thing?');
  });

  test('no text, or only spaces, means the submit goes ahead without asking', () => {
    expect(confirmMessage(undefined, undefined)).toBeNull();
    expect(confirmMessage('  ', undefined)).toBeNull();
  });
});

describe('ConfirmGate', () => {
  test('a form must be approved before it submits', () => {
    const gate = new ConfirmGate<object>();
    const form = {};
    expect(gate.pass(form)).toBe(false);
    // Answering No approves nothing, so the next submit is stopped again.
    expect(gate.pass(form)).toBe(false);
  });

  test('an approval lets exactly one submit through', () => {
    const gate = new ConfirmGate<object>();
    const form = {};
    gate.approve(form);
    expect(gate.pass(form)).toBe(true);
    expect(gate.pass(form)).toBe(false);
  });

  test('approving one form does not let another through', () => {
    const gate = new ConfirmGate<object>();
    const a = {};
    const b = {};
    gate.approve(a);
    expect(gate.pass(b)).toBe(false);
    expect(gate.pass(a)).toBe(true);
  });
});

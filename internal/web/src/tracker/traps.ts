/** What a trap's Trigger button does, by kind; mirrors trapTriggers in internal/tracker/traps.go. */

export interface TriggerButton {
  label: string;
  title: string;
}

const BUTTONS: Record<string, TriggerButton> = {
  pit: { label: 'Trigger (stays)', title: 'Marks it triggered; the pit stays on the board for the rest of the quest.' },
  long_pit: { label: 'Trigger (stays)', title: 'Marks it triggered; the pit stays on the board for the rest of the quest.' },
  falling_block: { label: 'Trigger (blocks the square)', title: 'The trap goes and its square becomes blocked squares.' },
  spear: { label: 'Trigger (gone)', title: 'The spear strikes and the trap is gone.' },
  boulder: {
    label: 'Trigger (rolls)',
    title: 'Marks it triggered. Move the boulder to where it stops and press "Turn into blocked squares", or block that square with "Block square".',
  },
};

export function triggerButton(kind: string): TriggerButton {
  return BUTTONS[kind] ?? { label: 'Trigger', title: 'Marks it triggered; keep it or remove it as the quest needs.' };
}

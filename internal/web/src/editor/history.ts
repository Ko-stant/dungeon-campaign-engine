/** Undo/redo over immutable snapshots, with a "saved" marker for dirty tracking. */
export class History<T> {
  #states: T[];
  #index = 0;
  #savedIndex = 0;
  readonly #limit: number;

  constructor(initial: T, limit = 200) {
    this.#states = [initial];
    this.#limit = limit;
  }

  get current(): T {
    return this.#states[this.#index] as T;
  }

  get canUndo(): boolean {
    return this.#index > 0;
  }

  get canRedo(): boolean {
    return this.#index < this.#states.length - 1;
  }

  /** True when the current state differs from the last saved one. */
  get dirty(): boolean {
    return this.#index !== this.#savedIndex;
  }

  push(state: T): void {
    if (state === this.current) {
      return;
    }
    this.#states = [...this.#states.slice(0, this.#index + 1), state];
    this.#index = this.#states.length - 1;
    const excess = this.#states.length - (this.#limit + 1);
    if (excess > 0) {
      this.#states = this.#states.slice(excess);
      this.#index -= excess;
      this.#savedIndex -= excess;
    }
    if (this.#savedIndex > this.#index) {
      this.#savedIndex = -1; // the saved state was on the discarded redo branch
    }
  }

  undo(): T {
    if (this.canUndo) {
      this.#index--;
    }
    return this.current;
  }

  redo(): T {
    if (this.canRedo) {
      this.#index++;
    }
    return this.current;
  }

  markSaved(): void {
    this.#savedIndex = this.#index;
  }
}

import { describe, it, expect, beforeEach } from 'vitest';
import { focusTrap } from './focus-trap';

describe('focusTrap', () => {
  let container: HTMLElement;

  beforeEach(() => {
    document.body.innerHTML = '<button id="outside">Outside</button><div id="trap"><button id="first">First</button><input id="middle"/><button id="last">Last</button></div>';
    container = document.getElementById('trap')!;
  });

  it('moves focus to first focusable element on activate', () => {
    focusTrap(container, true);
    expect(document.activeElement?.id).toBe('first');
  });

  it('Tab on last element wraps to first', () => {
    focusTrap(container, true);
    const last = document.getElementById('last') as HTMLButtonElement;
    last.focus();
    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true });
    container.dispatchEvent(event);
    expect(document.activeElement?.id).toBe('first');
  });

  it('Shift+Tab on first wraps to last', () => {
    focusTrap(container, true);
    const event = new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true });
    container.dispatchEvent(event);
    expect(document.activeElement?.id).toBe('last');
  });

  it('returns a teardown function that removes listeners', () => {
    const teardown = focusTrap(container, true);
    teardown();
    document.getElementById('outside')!.focus();
    expect(document.activeElement?.id).toBe('outside');
  });
});

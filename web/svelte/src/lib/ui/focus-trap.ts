const FOCUSABLE = 'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])';

export function focusTrap(container: HTMLElement, active: boolean): () => void {
  if (!active) return () => {};

  const focusables = () => Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE));

  const handleKeydown = (e: KeyboardEvent) => {
    if (e.key !== 'Tab') return;
    const elements = focusables();
    if (elements.length === 0) return;
    const first = elements[0];
    const last = elements[elements.length - 1];

    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };

  container.addEventListener('keydown', handleKeydown);
  focusables()[0]?.focus();

  return () => container.removeEventListener('keydown', handleKeydown);
}

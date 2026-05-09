<script lang="ts">
  import type { Snippet } from 'svelte';
  import { focusTrap } from './focus-trap';

  type Height = 'half' | 'full';

  interface Props {
    open?: boolean;
    height?: Height;
    onclose?: () => void;
    children?: Snippet;
  }

  let { open = $bindable(false), height = 'half', onclose, children }: Props = $props();

  let dialogEl: HTMLDivElement | undefined = $state();
  let dragging = $state(false);
  let dragOffset = $state(0);
  let sheetHeight = $state<Height>('half');
  let closing = $state(false);

  let velocitySamples: { y: number; t: number }[] = [];
  let dragStartY = 0;

  $effect(() => {
    if (open) {
      sheetHeight = height;
      dragOffset = 0;
      closing = false;
    }
  });

  $effect(() => {
    if (typeof document === 'undefined') return;
    if (open) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => { document.body.style.overflow = ''; };
  });

  $effect(() => {
    if (!open || !dialogEl) return;
    const teardown = focusTrap(dialogEl, true);
    return teardown;
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) onclose?.();
  }

  function onPointerDown(e: PointerEvent) {
    dragging = true;
    dragStartY = e.clientY;
    dragOffset = 0;
    velocitySamples = [{ y: e.clientY, t: e.timeStamp }];
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }

  function onPointerMove(e: PointerEvent) {
    if (!dragging) return;
    const dy = e.clientY - dragStartY;
    if (sheetHeight === 'half') {
      dragOffset = Math.max(-60, dy);
    } else {
      dragOffset = dy;
    }
    velocitySamples.push({ y: e.clientY, t: e.timeStamp });
    if (velocitySamples.length > 3) velocitySamples.shift();
  }

  function onPointerUp(_e: PointerEvent) {
    if (!dragging) return;
    dragging = false;

    let velocity = 0;
    if (velocitySamples.length >= 2) {
      const first = velocitySamples[0];
      const last = velocitySamples[velocitySamples.length - 1];
      const dt = last.t - first.t;
      if (dt > 0) velocity = (last.y - first.y) / dt;
    }

    if (sheetHeight === 'half') {
      if (dragOffset > 120 || velocity > 0.5) {
        dismiss();
      } else if (dragOffset < -60 || velocity < -0.5) {
        sheetHeight = 'full';
        dragOffset = 0;
      } else {
        dragOffset = 0;
      }
    } else {
      if (dragOffset > 120 || velocity > 0.5) {
        dismiss();
      } else if (dragOffset > 60) {
        sheetHeight = 'half';
        dragOffset = 0;
      } else {
        dragOffset = 0;
      }
    }

    velocitySamples = [];
  }

  function onPointerCancel(_e: PointerEvent) {
    dragging = false;
    dragOffset = 0;
    velocitySamples = [];
  }

  function dismiss() {
    closing = true;
    dragOffset = 0;
  }

  function onTransitionEnd(e: TransitionEvent) {
    if (closing && e.propertyName === 'transform') {
      closing = false;
      onclose?.();
    }
  }

  function getSheetStyle(): string {
    if (closing) {
      return 'transform: translateY(100%); transition: transform 0.3s cubic-bezier(0.32, 0.72, 0, 1);';
    }
    if (dragging) {
      return `transform: translateY(${Math.max(0, dragOffset)}px); transition: none;`;
    }
    return 'transition: transform 0.3s cubic-bezier(0.32, 0.72, 0, 1);';
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <button
    type="button"
    data-bottom-sheet-overlay
    onclick={dismiss}
    class="fixed inset-0 z-40 bg-black/50"
    aria-label="Close"
  ></button>

  <div
    role="dialog"
    aria-modal="true"
    bind:this={dialogEl}
    data-height={sheetHeight}
    class="fixed bottom-0 left-0 right-0 z-50 bg-background border-t border-border rounded-t-2xl shadow-2xl flex flex-col {sheetHeight === 'full' ? 'h-92dvh' : 'h-55dvh'}"
    style={getSheetStyle()}
    ontransitionend={onTransitionEnd}
  >
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      data-bottom-sheet-handle
      class="shrink-0 flex justify-center py-3 cursor-grab touch-none"
      onpointerdown={onPointerDown}
      onpointermove={onPointerMove}
      onpointerup={onPointerUp}
      onpointercancel={onPointerCancel}
    >
      <div class="w-10 h-1 rounded-full bg-muted-foreground/30"></div>
    </div>

    {@render children?.()}
  </div>
{/if}

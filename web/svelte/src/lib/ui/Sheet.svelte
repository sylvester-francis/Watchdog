<script lang="ts">
  import type { Snippet } from 'svelte';
  import { focusTrap } from './focus-trap';

  type Side = 'right' | 'left' | 'bottom';

  interface Props {
    open?: boolean;
    side?: Side;
    onclose?: () => void;
    children?: Snippet;
  }

  let { open = $bindable(false), side = 'right', onclose, children }: Props = $props();

  let dialogEl: HTMLDivElement | undefined = $state();

  $effect(() => {
    if (!open || !dialogEl) return;
    const teardown = focusTrap(dialogEl, true);
    return teardown;
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) onclose?.();
  }

  const sideClasses: Record<Side, string> = {
    right: 'inset-y-0 right-0 w-96 border-l',
    left:  'inset-y-0 left-0 w-96 border-r',
    bottom: 'inset-x-0 bottom-0 h-96 border-t',
  };
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div data-sheet-overlay onclick={() => onclose?.()} class="fixed inset-0 z-40 bg-black/40"></div>
  <div
    role="dialog"
    aria-modal="true"
    bind:this={dialogEl}
    data-side={side}
    class="fixed z-50 bg-card border-border p-4 shadow-lg overflow-y-auto {sideClasses[side]}"
  >
    {@render children?.()}
  </div>
{/if}

<script lang="ts">
  import type { Snippet } from 'svelte';
  import { focusTrap } from './focus-trap';

  interface Props {
    open?: boolean;
    onclose?: () => void;
    children?: Snippet;
  }

  let { open = $bindable(false), onclose, children }: Props = $props();

  let dialogEl: HTMLDivElement | undefined = $state();

  $effect(() => {
    if (!open || !dialogEl) return;
    const teardown = focusTrap(dialogEl, true);
    return teardown;
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) {
      onclose?.();
    }
  }

  function handleOverlayClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onclose?.();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div
    role="dialog"
    aria-modal="true"
    bind:this={dialogEl}
    onclick={handleOverlayClick}
    class="fixed inset-0 z-50 grid place-items-center bg-black/50 backdrop-blur-sm"
  >
    <div data-modal-content class="bg-card border border-border rounded-lg p-6 max-w-md w-full mx-4 shadow-lg">
      {@render children?.()}
    </div>
  </div>
{/if}

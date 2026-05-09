<script lang="ts">
  import type { Snippet } from 'svelte';
  import { focusTrap } from './focus-trap';

  type Size = 'sm' | 'md' | 'lg' | 'xl';

  interface Props {
    open?: boolean;
    onclose?: () => void;
    size?: Size;
    children?: Snippet;
  }

  let { open = $bindable(false), onclose, size = 'md', children }: Props = $props();

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

  const sizeClasses: Record<Size, string> = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
  };
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
    <div data-modal-content class="bg-card border border-border rounded-lg p-6 w-full mx-4 shadow-lg max-h-[90vh] overflow-y-auto {sizeClasses[size]}">
      {@render children?.()}
    </div>
  </div>
{/if}

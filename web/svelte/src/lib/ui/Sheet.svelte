<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    open?: boolean;
    side?: 'right' | 'left' | 'bottom';
    onclose?: () => void;
    children?: Snippet;
  }

  let { open = $bindable(false), side = 'right', onclose, children }: Props = $props();
</script>

{#if open}
  <div role="dialog" data-side={side} class="fixed inset-y-0 right-0 w-96 bg-card border-l border-border p-4">
    {@render children?.()}
  </div>
{/if}

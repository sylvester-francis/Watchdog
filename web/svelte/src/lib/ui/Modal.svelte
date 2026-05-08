<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    open?: boolean;
    onclose?: () => void;
    children?: Snippet;
  }

  let { open = $bindable(false), onclose, children }: Props = $props();
</script>

{#if open}
  <div role="dialog" class="fixed inset-0 grid place-items-center bg-black/50">
    <div class="bg-card border border-border rounded-lg p-6 max-w-md">
      {@render children?.()}
    </div>
  </div>
{/if}

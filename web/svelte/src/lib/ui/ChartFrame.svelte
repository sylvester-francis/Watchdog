<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string;
    height?: string;
    loading?: boolean;
    children?: Snippet;
  }

  let { title, height = '240px', loading = false, children }: Props = $props();
</script>

<div class="border border-border rounded-lg p-3" style:height>
  {#if title}<h4 class="text-sm text-muted-foreground mb-2">{title}</h4>{/if}
  {#if loading}
    <div class="bg-muted rounded animate-pulse h-full"></div>
  {:else}
    {@render children?.()}
  {/if}
</div>

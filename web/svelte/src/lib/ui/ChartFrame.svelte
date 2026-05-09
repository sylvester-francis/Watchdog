<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string;
    subtitle?: string;
    height?: string;
    loading?: boolean;
    children?: Snippet;
  }

  let { title, subtitle, height = '240px', loading = false, children }: Props = $props();
</script>

<div class="border border-border rounded-lg p-3 bg-card flex flex-col">
  {#if title || subtitle}
    <div class="mb-2">
      {#if title}<h4 class="text-sm font-medium text-foreground">{title}</h4>{/if}
      {#if subtitle}<p class="text-xs text-muted-foreground">{subtitle}</p>{/if}
    </div>
  {/if}
  <div style:height>
    {#if loading}
      <div class="bg-muted rounded animate-pulse h-full"></div>
    {:else}
      {@render children?.()}
    {/if}
  </div>
</div>

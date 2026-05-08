<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    value: string | number;
    delta?: string;
    children?: Snippet;
  }

  let { label, value, delta, children }: Props = $props();
</script>

<div class="flex flex-col gap-1 p-3 rounded-lg border border-border">
  <span class="text-xs text-muted-foreground">{label}</span>
  <span class="text-stat">{value}</span>
  {#if delta}<span class="text-xs text-muted-foreground">{delta}</span>{/if}
  {@render children?.()}
</div>

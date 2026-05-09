<script lang="ts">
  import type { Snippet } from 'svelte';

  type DeltaDirection = 'up' | 'down' | 'neutral';

  interface Props {
    label: string;
    value: string | number;
    delta?: string;
    deltaDirection?: DeltaDirection;
    children?: Snippet;
  }

  let { label, value, delta, deltaDirection = 'neutral', children }: Props = $props();

  const deltaClasses: Record<DeltaDirection, string> = {
    up:      'text-status-up',
    down:    'text-destructive',
    neutral: 'text-muted-foreground',
  };
</script>

<div class="flex flex-col gap-1 p-3 rounded-lg border border-border bg-card">
  <span class="text-xs text-muted-foreground uppercase tracking-wider">{label}</span>
  <span class="text-stat text-foreground">{value}</span>
  {#if delta}
    <span data-delta={deltaDirection} class="text-xs {deltaClasses[deltaDirection]}">{delta}</span>
  {/if}
  {@render children?.()}
</div>

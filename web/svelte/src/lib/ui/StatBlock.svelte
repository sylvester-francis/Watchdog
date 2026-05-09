<script lang="ts">
  import type { Snippet } from 'svelte';

  type DeltaDirection = 'up' | 'down' | 'neutral';
  type Accent = 'up' | 'down' | 'warn' | 'neutral';

  interface Props {
    label: string;
    value: string | number;
    delta?: string;
    deltaDirection?: DeltaDirection;
    accent?: Accent;
    children?: Snippet;
  }

  let { label, value, delta, deltaDirection = 'neutral', accent, children }: Props = $props();

  const deltaClasses: Record<DeltaDirection, string> = {
    up:      'text-status-up',
    down:    'text-destructive',
    neutral: 'text-muted-foreground',
  };

  const accentClasses: Record<Accent, string> = {
    up:      'border-l-[3px] border-l-status-up',
    down:    'border-l-[3px] border-l-destructive',
    warn:    'border-l-[3px] border-l-warning',
    neutral: 'border-l-[3px] border-l-muted',
  };
</script>

<div data-accent={accent} class="flex flex-col gap-1 p-3 rounded-lg border border-border bg-card {accent ? accentClasses[accent] : ''}">
  <span class="text-xs text-muted-foreground uppercase tracking-wider">{label}</span>
  <span class="text-stat text-foreground">{value}</span>
  {#if delta}
    <span data-delta={deltaDirection} class="text-xs {deltaClasses[deltaDirection]}">{delta}</span>
  {/if}
  {@render children?.()}
</div>

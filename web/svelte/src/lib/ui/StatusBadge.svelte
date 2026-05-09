<script lang="ts">
  import type { Snippet } from 'svelte';

  type Status = 'up' | 'down' | 'warn' | 'expired' | 'maintenance' | 'unknown';

  interface Props {
    status: Status;
    children?: Snippet;
  }

  let { status, children }: Props = $props();

  const glyph: Record<Status, string> = {
    up: '●',
    down: '✕',
    warn: '⚠',
    expired: '○',
    maintenance: '⏸',
    unknown: '?',
  };
</script>

<span data-status={status} class="ui-status-badge inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs">
  <span aria-hidden="true">{glyph[status]}</span>
  {@render children?.()}
</span>

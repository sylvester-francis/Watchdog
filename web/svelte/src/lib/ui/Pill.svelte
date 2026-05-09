<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    tone?: 'neutral' | 'accent' | 'up' | 'down' | 'warn';
    children?: Snippet;
  }

  let { tone = 'neutral', children }: Props = $props();
</script>

<span data-tone={tone} class="ui-pill inline-flex items-center px-2 py-0.5 rounded-full text-xs border">
  {@render children?.()}
</span>

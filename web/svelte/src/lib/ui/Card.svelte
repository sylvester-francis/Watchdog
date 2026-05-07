<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'default' | 'elevated' | 'accent';
    children?: Snippet;
  }

  let { variant = 'default', children }: Props = $props();
</script>

<div data-variant={variant} class="bg-card border border-border rounded-lg p-4">
  {@render children?.()}
</div>

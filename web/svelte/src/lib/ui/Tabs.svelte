<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    value?: string;
    children?: Snippet;
  }

  let { value = $bindable(''), children }: Props = $props();
</script>

<div role="tablist" data-active={value} class="flex gap-2 border-b border-border">
  {@render children?.()}
</div>

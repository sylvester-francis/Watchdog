<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    children?: Snippet;
  }

  let { label, children }: Props = $props();
</script>

<span title={label}>
  {@render children?.()}
</span>

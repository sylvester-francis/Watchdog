<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string;
    description?: string;
    children?: Snippet;
  }

  let { title, description, children }: Props = $props();
</script>

<div class="flex flex-col items-center gap-2 py-12 text-center">
  {#if title}<h3 class="text-foreground">{title}</h3>{/if}
  {#if description}<p class="text-muted-foreground">{description}</p>{/if}
  {@render children?.()}
</div>

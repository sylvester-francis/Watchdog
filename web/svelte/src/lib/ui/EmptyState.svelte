<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string;
    description?: string;
    icon?: Snippet;
    cta?: Snippet;
    children?: Snippet;
  }

  let { title, description, icon, cta, children }: Props = $props();
</script>

<div class="flex flex-col items-center gap-3 py-12 text-center">
  {#if icon}
    <div class="text-muted-foreground">
      {@render icon()}
    </div>
  {/if}
  {#if title}<h3 class="text-foreground font-medium">{title}</h3>{/if}
  {#if description}<p class="text-sm text-muted-foreground max-w-md">{description}</p>{/if}
  {@render children?.()}
  {#if cta}
    <div class="mt-2">
      {@render cta()}
    </div>
  {/if}
</div>

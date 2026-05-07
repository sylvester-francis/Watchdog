<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label?: string;
    error?: string | null;
    htmlFor?: string;
    children?: Snippet;
  }

  let { label, error = null, htmlFor, children }: Props = $props();
</script>

<div class="flex flex-col gap-1">
  {#if label}
    <label for={htmlFor} class="text-sm text-muted-foreground">{label}</label>
  {/if}
  {@render children?.()}
  {#if error}
    <p class="text-sm text-destructive">{error}</p>
  {/if}
</div>

<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label?: string;
    error?: string | null;
    required?: boolean;
    htmlFor?: string;
    children?: Snippet;
  }

  let { label, error = null, required = false, htmlFor, children }: Props = $props();
</script>

<div class="flex flex-col gap-1">
  {#if label}
    <label for={htmlFor} class="text-sm text-muted-foreground">
      {label}{#if required}<span class="text-destructive ml-0.5" aria-hidden="true">*</span>{/if}
    </label>
  {/if}
  {@render children?.()}
  {#if error}
    <p role="alert" class="text-sm text-destructive">{error}</p>
  {/if}
</div>

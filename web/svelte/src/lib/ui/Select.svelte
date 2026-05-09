<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    value?: string;
    disabled?: boolean;
    error?: boolean;
    name?: string;
    id?: string;
    children?: Snippet;
  }

  let {
    value = $bindable(''),
    disabled = false,
    error = false,
    name,
    id,
    children,
  }: Props = $props();
</script>

<select
  bind:value
  {disabled}
  {name}
  {id}
  aria-invalid={error || undefined}
  class="w-full bg-card-elevated border rounded px-3 py-2 text-foreground transition-colors
         focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background
         disabled:opacity-50 disabled:cursor-not-allowed
         {error ? 'border-destructive' : 'border-border'}"
>
  {@render children?.()}
</select>

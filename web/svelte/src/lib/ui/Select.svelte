<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    value?: string;
    disabled?: boolean;
    name?: string;
    id?: string;
    children?: Snippet;
  }

  let {
    value = $bindable(''),
    disabled = false,
    name,
    id,
    children,
  }: Props = $props();
</script>

<select bind:value {disabled} {name} {id}
        class="bg-card-elevated border border-border rounded px-3 py-2 text-foreground">
  {@render children?.()}
</select>

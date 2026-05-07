<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'primary' | 'secondary' | 'ghost' | 'destructive' | 'outline';
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    onclick?: (e: MouseEvent) => void;
    children?: Snippet;
  }

  let { variant = 'primary', type = 'button', disabled = false, onclick, children }: Props = $props();
</script>

<button {type} {disabled} {onclick} data-variant={variant}
        class="px-4 py-2 rounded bg-accent text-accent-foreground disabled:opacity-50">
  {@render children?.()}
</button>

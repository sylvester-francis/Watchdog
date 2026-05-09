<script lang="ts">
  import type { Snippet } from 'svelte';

  type Variant = 'default' | 'elevated' | 'accent';

  interface Props {
    variant?: Variant;
    children?: Snippet;
  }

  let { variant = 'default', children }: Props = $props();

  const variantClasses: Record<Variant, string> = {
    default: 'bg-card border border-border',
    elevated: 'bg-card-elevated border border-border shadow',
    accent: 'bg-card border border-accent',
  };
</script>

<div data-variant={variant} class="rounded-lg p-4 {variantClasses[variant]}">
  {@render children?.()}
</div>

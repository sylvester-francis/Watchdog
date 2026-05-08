<script lang="ts">
  import type { Snippet } from 'svelte';

  type Variant = 'primary' | 'secondary' | 'ghost' | 'destructive' | 'outline';
  type Size = 'sm' | 'md' | 'lg';

  interface Props {
    variant?: Variant;
    size?: Size;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    onclick?: (e: MouseEvent) => void;
    children?: Snippet;
  }

  let { variant = 'primary', size = 'md', type = 'button', disabled = false, onclick, children }: Props = $props();

  const variantClasses: Record<Variant, string> = {
    primary: 'bg-accent text-accent-foreground hover:opacity-90',
    secondary: 'bg-card-elevated text-foreground border border-border hover:bg-muted',
    ghost: 'bg-transparent text-foreground hover:bg-card-elevated',
    destructive: 'bg-destructive text-destructive-foreground hover:opacity-90',
    outline: 'bg-transparent text-foreground border border-border hover:bg-card-elevated',
  };

  const sizeClasses: Record<Size, string> = {
    sm: 'px-3 py-1 text-sm rounded',
    md: 'px-4 py-2 text-base rounded',
    lg: 'px-6 py-3 text-lg rounded-lg',
  };
</script>

<button
  {type}
  {disabled}
  {onclick}
  data-variant={variant}
  data-size={size}
  class="inline-flex items-center justify-center font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background {variantClasses[variant]} {sizeClasses[size]}"
>
  {@render children?.()}
</button>

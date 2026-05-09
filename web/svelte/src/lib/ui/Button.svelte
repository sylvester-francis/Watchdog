<script lang="ts">
  import type { Snippet } from 'svelte';

  type Variant = 'primary' | 'secondary' | 'ghost' | 'destructive' | 'outline';
  type Size = 'sm' | 'md' | 'lg';
  type Tone = 'accent' | 'up' | 'warn' | 'down';

  interface Props {
    variant?: Variant;
    size?: Size;
    tone?: Tone;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    onclick?: (e: MouseEvent) => void;
    children?: Snippet;
  }

  let { variant = 'primary', size = 'md', tone, type = 'button', disabled = false, onclick, children }: Props = $props();

  const variantClasses: Record<Variant, string> = {
    primary: 'bg-accent text-accent-foreground hover:opacity-90',
    secondary: 'bg-card-elevated text-foreground border border-border hover:bg-muted',
    ghost: 'bg-transparent text-foreground hover:bg-card-elevated',
    destructive: 'bg-destructive text-destructive-foreground hover:opacity-90',
    outline: 'bg-transparent text-foreground border border-border hover:bg-card-elevated',
  };

  const toneClasses: Record<Tone, string> = {
    accent: 'bg-accent/10 text-accent hover:bg-accent/20',
    up:     'bg-success/10 text-success hover:bg-success/20',
    warn:   'bg-warning/10 text-warning hover:bg-warning/20',
    down:   'bg-destructive/10 text-destructive hover:bg-destructive/20',
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
  data-tone={tone}
  class="inline-flex items-center justify-center font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background {tone ? toneClasses[tone] : variantClasses[variant]} {sizeClasses[size]}"
>
  {@render children?.()}
</button>

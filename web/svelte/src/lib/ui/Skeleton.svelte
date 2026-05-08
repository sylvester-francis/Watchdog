<script lang="ts">
  type Variant = 'text' | 'chart' | 'tableRow' | 'card';

  interface Props {
    variant?: Variant;
    width?: string;
    height?: string;
  }

  const defaults: Record<Variant, { width: string; height: string }> = {
    text:     { width: '100%', height: '1rem' },
    chart:    { width: '100%', height: '240px' },
    tableRow: { width: '100%', height: '40px' },
    card:     { width: '100%', height: '120px' },
  };

  let { variant = 'text', width, height }: Props = $props();

  const finalWidth = $derived(width ?? defaults[variant].width);
  const finalHeight = $derived(height ?? defaults[variant].height);
</script>

<div data-variant={variant} style:width={finalWidth} style:height={finalHeight} class="bg-muted rounded animate-pulse"></div>

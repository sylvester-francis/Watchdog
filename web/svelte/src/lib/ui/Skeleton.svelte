<script lang="ts">
  interface Props {
    variant?: 'text' | 'chart' | 'tableRow' | 'card';
    width?: string;
    height?: string;
  }

  let { variant = 'text', width = '100%', height = '1rem' }: Props = $props();
</script>

<div data-variant={variant} style:width style:height class="bg-muted rounded animate-pulse"></div>

<script lang="ts">
  type Status = 'up' | 'down' | 'warn' | 'unknown';

  interface Props {
    data: number[];
    color?: string;
    status?: Status;
    fill?: boolean;
    width?: number;
    height?: number;
  }

  let { data, color, status, fill = false, width = 100, height = 18 }: Props = $props();

  const stroke = $derived.by(() => {
    if (color) return color;
    if (status === 'up') return 'var(--status-up)';
    if (status === 'down') return 'var(--status-down)';
    if (status === 'warn') return 'var(--status-warn)';
    return 'currentColor';
  });

  const points = $derived.by(() => {
    if (data.length === 0) return '';
    const max = Math.max(...data);
    const min = Math.min(...data);
    const range = max - min || 1;
    return data
      .map((v, i) => {
        const x = (i / (data.length - 1 || 1)) * width;
        const y = height - ((v - min) / range) * height;
        return `${x},${y}`;
      })
      .join(' ');
  });

  const polygonPoints = $derived(points ? `0,${height} ${points} ${width},${height}` : '');
</script>

<svg viewBox="0 0 {width} {height}" preserveAspectRatio="none" data-status={status}
     style:width="100%" style:height="{height}px">
  {#if fill && polygonPoints}
    <polygon points={polygonPoints} fill={stroke} opacity="0.15" />
  {/if}
  <polyline fill="none" stroke={stroke} stroke-width="1.2" points={points} />
</svg>

<script lang="ts">
  interface Props {
    data: number[];
    color?: string;
    width?: number;
    height?: number;
  }

  let { data, color = 'currentColor', width = 100, height = 18 }: Props = $props();

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
</script>

<svg viewBox="0 0 {width} {height}" preserveAspectRatio="none" style:width="100%" style:height="{height}px">
  <polyline fill="none" stroke={color} stroke-width="1.2" points={points} />
</svg>

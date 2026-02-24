<script lang="ts">
	interface Props {
		values: number[];
		color?: string;
	}

	let { values, color = '#22c55e' }: Props = $props();

	const width = 56;
	const height = 20;
	const padding = 1;

	let points = $derived(() => {
		if (values.length === 0) return '';
		const min = Math.min(...values);
		const max = Math.max(...values);
		const range = max - min || 1;
		const usableW = width - padding * 2;
		const usableH = height - padding * 2;

		return values
			.map((v, i) => {
				const x = padding + (i / Math.max(values.length - 1, 1)) * usableW;
				const y = padding + usableH - ((v - min) / range) * usableH;
				return `${x},${y}`;
			})
			.join(' ');
	});
</script>

<svg {width} {height} class="overflow-visible">
	{#if values.length > 0}
		<polyline
			points={points()}
			fill="none"
			stroke={color}
			stroke-width="1.5"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	{/if}
</svg>

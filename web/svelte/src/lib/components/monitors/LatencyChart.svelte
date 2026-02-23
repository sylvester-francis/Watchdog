<script lang="ts">
	import { onMount } from 'svelte';
	import { Loader2, Timer } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { LatencyPoint } from '$lib/types';

	interface Props {
		monitorId: string;
	}

	let { monitorId }: Props = $props();

	const toast = getToasts();

	type Period = '1h' | '24h' | '7d' | '30d';
	const periods: { value: Period; label: string }[] = [
		{ value: '1h', label: '1H' },
		{ value: '24h', label: '24H' },
		{ value: '7d', label: '7D' },
		{ value: '30d', label: '30D' }
	];

	let activePeriod = $state<Period>('24h');
	let data = $state<LatencyPoint[]>([]);
	let loading = $state(true);

	// SVG dimensions
	const svgWidth = 800;
	const svgHeight = 200;
	const paddingLeft = 55;
	const paddingRight = 20;
	const paddingTop = 15;
	const paddingBottom = 30;
	const chartWidth = svgWidth - paddingLeft - paddingRight;
	const chartHeight = svgHeight - paddingTop - paddingBottom;

	// Derived chart calculations
	let values = $derived(data.map((d) => d.avg_ms));
	let maxVal = $derived(values.length > 0 ? Math.max(...values) : 0);
	let minVal = $derived(values.length > 0 ? Math.min(...values) : 0);
	let range = $derived(maxVal - minVal || 1);
	// Add 10% padding to y-axis
	let yMax = $derived(maxVal + range * 0.1);
	let yMin = $derived(Math.max(0, minVal - range * 0.1));
	let yRange = $derived(yMax - yMin || 1);

	// Y-axis grid lines (5 levels)
	let yTicks = $derived.by(() => {
		const ticks: number[] = [];
		for (let i = 0; i <= 4; i++) {
			ticks.push(yMin + (yRange * i) / 4);
		}
		return ticks;
	});

	// X-axis labels (show ~6 evenly distributed)
	let xLabels = $derived.by(() => {
		if (data.length === 0) return [];
		const count = Math.min(6, data.length);
		const step = Math.max(1, Math.floor((data.length - 1) / (count - 1)));
		const labels: { x: number; text: string }[] = [];
		for (let i = 0; i < data.length; i += step) {
			const d = new Date(data[i].time);
			let text: string;
			if (activePeriod === '1h' || activePeriod === '24h') {
				text = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
			} else {
				text = d.toLocaleDateString([], { month: 'short', day: 'numeric' });
			}
			const x = paddingLeft + (i / Math.max(data.length - 1, 1)) * chartWidth;
			labels.push({ x, text });
		}
		return labels;
	});

	// SVG polyline points for the line
	let linePath = $derived.by(() => {
		if (data.length === 0) return '';
		return data
			.map((d, i) => {
				const x = paddingLeft + (i / Math.max(data.length - 1, 1)) * chartWidth;
				const y = paddingTop + chartHeight - ((d.avg_ms - yMin) / yRange) * chartHeight;
				return `${x},${y}`;
			})
			.join(' ');
	});

	// SVG polygon points for the fill area (line + bottom edge closed)
	let fillPath = $derived.by(() => {
		if (data.length === 0) return '';
		const linePoints = data.map((d, i) => {
			const x = paddingLeft + (i / Math.max(data.length - 1, 1)) * chartWidth;
			const y = paddingTop + chartHeight - ((d.avg_ms - yMin) / yRange) * chartHeight;
			return `${x},${y}`;
		});
		const bottomRight = `${paddingLeft + chartWidth},${paddingTop + chartHeight}`;
		const bottomLeft = `${paddingLeft},${paddingTop + chartHeight}`;
		return [...linePoints, bottomRight, bottomLeft].join(' ');
	});

	function yToSvg(val: number): number {
		return paddingTop + chartHeight - ((val - yMin) / yRange) * chartHeight;
	}

	function formatMs(val: number): string {
		if (val >= 1000) return `${(val / 1000).toFixed(1)}s`;
		return `${Math.round(val)}ms`;
	}

	async function fetchData(period: Period) {
		loading = true;
		try {
			const res = await monitorsApi.getLatencyHistory(monitorId, period);
			data = res.data ?? [];
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load latency data');
			data = [];
		} finally {
			loading = false;
		}
	}

	function selectPeriod(period: Period) {
		activePeriod = period;
		fetchData(period);
	}

	onMount(() => {
		fetchData(activePeriod);
	});
</script>

<div class="bg-card border border-border rounded-lg">
	<!-- Header -->
	<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
		<div class="flex items-center space-x-2">
			<Timer class="w-4 h-4 text-accent" />
			<h3 class="text-sm font-medium text-foreground">Response Time</h3>
		</div>
		<div class="flex items-center space-x-1">
			{#each periods as p}
				<button
					onclick={() => selectPeriod(p.value)}
					class="px-2 py-1 text-[10px] font-medium rounded transition-colors {activePeriod === p.value
						? 'bg-accent/15 text-accent'
						: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
				>
					{p.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Chart area -->
	<div class="p-5">
		{#if loading}
			<div class="flex items-center justify-center h-[200px]">
				<Loader2 class="w-5 h-5 text-muted-foreground animate-spin" />
			</div>
		{:else if data.length === 0}
			<div class="flex items-center justify-center h-[200px]">
				<p class="text-xs text-muted-foreground">No latency data for this period</p>
			</div>
		{:else}
			<svg viewBox="0 0 {svgWidth} {svgHeight}" class="w-full" preserveAspectRatio="xMidYMid meet">
				<!-- Y-axis grid lines and labels -->
				{#each yTicks as tick}
					<line
						x1={paddingLeft}
						y1={yToSvg(tick)}
						x2={svgWidth - paddingRight}
						y2={yToSvg(tick)}
						stroke="currentColor"
						class="text-border"
						stroke-width="0.5"
						stroke-dasharray="4,4"
					/>
					<text
						x={paddingLeft - 8}
						y={yToSvg(tick) + 3}
						text-anchor="end"
						class="text-muted-foreground"
						fill="currentColor"
						font-size="10"
						font-family="ui-monospace, monospace"
					>
						{formatMs(tick)}
					</text>
				{/each}

				<!-- X-axis labels -->
				{#each xLabels as label}
					<text
						x={label.x}
						y={svgHeight - 5}
						text-anchor="middle"
						class="text-muted-foreground"
						fill="currentColor"
						font-size="10"
						font-family="ui-monospace, monospace"
					>
						{label.text}
					</text>
				{/each}

				<!-- Fill gradient area -->
				<defs>
					<linearGradient id="latency-fill-{monitorId}" x1="0" y1="0" x2="0" y2="1">
						<stop offset="0%" stop-color="#3b82f6" stop-opacity="0.15" />
						<stop offset="100%" stop-color="#3b82f6" stop-opacity="0.02" />
					</linearGradient>
				</defs>
				<polygon
					points={fillPath}
					fill="url(#latency-fill-{monitorId})"
				/>

				<!-- Line -->
				<polyline
					points={linePath}
					fill="none"
					stroke="#3b82f6"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
		{/if}
	</div>
</div>

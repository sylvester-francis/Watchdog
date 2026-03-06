<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2, Timer } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { LatencyPoint } from '$lib/types';
	import {
		Chart,
		LineController,
		LineElement,
		PointElement,
		LinearScale,
		CategoryScale,
		Filler,
		Tooltip,
		Legend
	} from 'chart.js';

	Chart.register(
		LineController,
		LineElement,
		PointElement,
		LinearScale,
		CategoryScale,
		Filler,
		Tooltip,
		Legend
	);

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
	let canvasEl = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let chart: Chart | null = null;

	function formatLabel(time: string, period: Period): string {
		const d = new Date(time);
		if (period === '1h' || period === '24h') {
			return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		}
		return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
	}

	function buildChart() {
		if (!canvasEl || data.length === 0) return;

		if (chart) {
			chart.destroy();
			chart = null;
		}

		const labels = data.map((p) => formatLabel(p.time, activePeriod));
		const avgData = data.map((p) => p.avg_ms);
		const minData = data.map((p) => p.min_ms);
		const maxData = data.map((p) => p.max_ms);

		chart = new Chart(canvasEl, {
			type: 'line',
			data: {
				labels,
				datasets: [
					{
						label: 'Avg',
						data: avgData,
						borderColor: '#3b82f6',
						backgroundColor: 'rgba(59, 130, 246, 0.08)',
						fill: true,
						tension: 0.3,
						borderWidth: 2,
						pointRadius: 1,
						pointHoverRadius: 4,
						pointBackgroundColor: '#3b82f6',
						pointHoverBackgroundColor: '#3b82f6'
					},
					{
						label: 'Max',
						data: maxData,
						borderColor: 'rgba(239, 68, 68, 0.5)',
						borderWidth: 1,
						borderDash: [3, 3],
						pointRadius: 0,
						pointHoverRadius: 3,
						pointHoverBackgroundColor: '#ef4444',
						fill: false
					},
					{
						label: 'Min',
						data: minData,
						borderColor: 'rgba(34, 197, 94, 0.5)',
						borderWidth: 1,
						borderDash: [3, 3],
						pointRadius: 0,
						pointHoverRadius: 3,
						pointHoverBackgroundColor: '#22c55e',
						fill: false
					}
				]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				interaction: {
					mode: 'index',
					intersect: false
				},
				plugins: {
					legend: {
						display: true,
						position: 'top',
						align: 'end',
						labels: {
							boxWidth: 12,
							padding: 12,
							font: { size: 10, family: "'Inter', system-ui, sans-serif" },
							color: '#a1a1aa',
							usePointStyle: true,
							pointStyle: 'line'
						}
					},
					tooltip: {
						backgroundColor: '#18181b',
						borderColor: '#27272a',
						borderWidth: 1,
						titleFont: { family: "'Inter', system-ui, sans-serif", size: 11 },
						bodyFont: { family: "'JetBrains Mono', ui-monospace, monospace", size: 12 },
						padding: 10,
						cornerRadius: 6,
						displayColors: true,
						boxWidth: 8,
						boxPadding: 4,
						callbacks: {
							label: (ctx: any) => ` ${ctx.dataset.label}: ${ctx.parsed.y}ms`
						}
					}
				},
				scales: {
					x: {
						grid: { color: '#27272a', lineWidth: 0.5 },
						ticks: {
							maxTicksLimit: 8,
							maxRotation: 0,
							color: '#71717a',
							font: { size: 10, family: "'JetBrains Mono', ui-monospace, monospace" }
						},
						border: { color: '#27272a' }
					},
					y: {
						grid: { color: '#27272a', lineWidth: 0.5 },
						ticks: {
							color: '#71717a',
							font: { size: 10, family: "'JetBrains Mono', ui-monospace, monospace" },
							callback: (v: any) => `${v}ms`
						},
						beginAtZero: true,
						border: { color: '#27272a' }
					}
				},
				animation: {
					duration: 600,
					easing: 'easeOutQuart'
				}
			}
		});
	}

	async function fetchData(period: Period) {
		loading = true;
		try {
			const res = await monitorsApi.getLatencyHistory(monitorId, period);
			data = Array.isArray(res) ? res : [];
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

	$effect(() => {
		if (!loading && data.length > 0 && canvasEl) {
			buildChart();
		}
	});

	onMount(() => {
		fetchData(activePeriod);
	});

	onDestroy(() => {
		if (chart) {
			chart.destroy();
			chart = null;
		}
	});
</script>

<div class="bg-card border border-border rounded-lg">
	<!-- Header -->
	<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
		<div>
			<div class="flex items-center space-x-2">
				<Timer class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">Response Time</h3>
			</div>
			<p class="text-[10px] text-muted-foreground mt-0.5 ml-6">How fast this target responds. Spikes mean slowdowns; flat lines are healthy.</p>
		</div>
		<div class="flex items-center space-x-1">
			{#each periods as p}
				<button
					onclick={() => selectPeriod(p.value)}
					class="px-2.5 py-1 text-[10px] font-medium rounded transition-colors {activePeriod === p.value
						? 'bg-foreground/[0.08] text-foreground'
						: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
				>
					{p.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Chart area -->
	<div class="px-5 py-4">
		{#if loading}
			<div class="flex items-center justify-center h-[220px]">
				<Loader2 class="w-5 h-5 text-muted-foreground animate-spin" />
			</div>
		{:else if data.length === 0}
			<div class="flex items-center justify-center h-[220px]">
				<p class="text-xs text-muted-foreground">No latency data for this period</p>
			</div>
		{:else}
			<div class="h-[220px]">
				<canvas bind:this={canvasEl}></canvas>
			</div>
		{/if}
	</div>
</div>

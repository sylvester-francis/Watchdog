<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2, Activity } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { HeartbeatPoint } from '$lib/types';
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
	let heartbeats = $state<HeartbeatPoint[]>([]);
	let loading = $state(true);
	let hasNumericValues = $state(false);
	let canvasEl = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let chart: Chart | null = null;
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	function parseNumericValue(errorMessage: string | undefined): number | null {
		if (!errorMessage) return null;
		const trimmed = errorMessage.trim();
		if (trimmed === '') return null;
		const num = Number(trimmed);
		if (isNaN(num)) return null;
		return num;
	}

	function formatLabel(time: string, period: Period): string {
		const d = new Date(time);
		if (period === '1h' || period === '24h') {
			return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		}
		return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
	}

	function buildChart() {
		if (!canvasEl) return;

		const points: { label: string; value: number }[] = [];
		for (const hb of heartbeats) {
			if (hb.status !== 'up') continue;
			const val = parseNumericValue(hb.error_message);
			if (val !== null) {
				points.push({ label: formatLabel(hb.time, activePeriod), value: val });
			}
		}

		hasNumericValues = points.length > 0;
		if (!hasNumericValues) return;

		if (chart) {
			chart.destroy();
			chart = null;
		}

		const labels = points.map((p) => p.label);
		const values = points.map((p) => p.value);

		chart = new Chart(canvasEl, {
			type: 'line',
			data: {
				labels,
				datasets: [
					{
						label: 'Value',
						data: values,
						borderColor: '#10b981',
						backgroundColor: 'rgba(16, 185, 129, 0.08)',
						fill: true,
						tension: 0.3,
						borderWidth: 2,
						pointRadius: 1,
						pointHoverRadius: 4,
						pointBackgroundColor: '#10b981',
						pointHoverBackgroundColor: '#10b981'
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
							label: (ctx: any) => ` ${ctx.dataset.label}: ${ctx.parsed.y}`
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
							font: { size: 10, family: "'JetBrains Mono', ui-monospace, monospace" }
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
			const res = await monitorsApi.getHeartbeats(monitorId, period);
			heartbeats = Array.isArray(res) ? res : [];
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load SNMP data');
			heartbeats = [];
		} finally {
			loading = false;
		}
	}

	function selectPeriod(period: Period) {
		activePeriod = period;
		fetchData(period);
	}

	function startPolling() {
		stopPolling();
		pollTimer = setInterval(() => {
			fetchData(activePeriod);
		}, 30_000);
	}

	function stopPolling() {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	$effect(() => {
		if (!loading && heartbeats.length > 0 && canvasEl) {
			buildChart();
		}
	});

	onMount(() => {
		fetchData(activePeriod);
		startPolling();
	});

	onDestroy(() => {
		stopPolling();
		if (chart) {
			chart.destroy();
			chart = null;
		}
	});
</script>

{#if loading || hasNumericValues || heartbeats.length === 0}
<div class="bg-card border border-border rounded-lg">
	<!-- Header -->
	<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
		<div>
			<div class="flex items-center space-x-2">
				<Activity class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">SNMP Value</h3>
			</div>
			<p class="text-[10px] text-muted-foreground mt-0.5 ml-6">Tracks the polled value over time. Useful for counters, gauges, and metrics.</p>
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
		{:else if heartbeats.length === 0}
			<div class="flex items-center justify-center h-[220px]">
				<p class="text-xs text-muted-foreground">No SNMP data for this period</p>
			</div>
		{:else}
			<div class="h-[220px]">
				<canvas bind:this={canvasEl}></canvas>
			</div>
		{/if}
	</div>
</div>
{/if}

<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2, Activity } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { HeartbeatPoint } from '$lib/types';
	import {
		Chart,
		BarController,
		BarElement,
		LinearScale,
		CategoryScale,
		Tooltip
	} from 'chart.js';

	Chart.register(BarController, BarElement, LinearScale, CategoryScale, Tooltip);

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
		if (!canvasEl || heartbeats.length === 0) return;

		if (chart) {
			chart.destroy();
			chart = null;
		}

		const labels = heartbeats.map((p) => formatLabel(p.time, activePeriod));
		const statusData = heartbeats.map((p) => (p.status === 'up' ? 1 : 0));
		const colors = heartbeats.map((p) =>
			p.status === 'up' ? 'rgba(34, 197, 94, 0.7)' : 'rgba(239, 68, 68, 0.7)'
		);

		chart = new Chart(canvasEl, {
			type: 'bar',
			data: {
				labels,
				datasets: [
					{
						label: 'Status',
						data: statusData,
						backgroundColor: colors,
						borderRadius: 2,
						barPercentage: 0.9,
						categoryPercentage: 0.95
					}
				]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: { display: false },
					tooltip: {
						backgroundColor: '#18181b',
						borderColor: '#27272a',
						borderWidth: 1,
						titleFont: { family: "'Inter', system-ui, sans-serif", size: 11 },
						bodyFont: { family: "'JetBrains Mono', ui-monospace, monospace", size: 12 },
						padding: 10,
						cornerRadius: 6,
						callbacks: {
							label: (ctx: any) => {
								const hb = heartbeats[ctx.dataIndex];
								const status = hb.status === 'up' ? 'Running' : 'Down';
								return ` ${status}`;
							},
							afterLabel: (ctx: any) => {
								const hb = heartbeats[ctx.dataIndex];
								return hb.error_message ? ` ${hb.error_message}` : '';
							}
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
						display: false,
						min: 0,
						max: 1.2
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
			toast.error(err instanceof Error ? err.message : 'Failed to load status data');
			heartbeats = [];
		} finally {
			loading = false;
		}
	}

	function selectPeriod(period: Period) {
		activePeriod = period;
		fetchData(period);
	}

	$effect(() => {
		if (!loading && heartbeats.length > 0 && canvasEl) {
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
		<div class="flex items-center space-x-2">
			<Activity class="w-4 h-4 text-muted-foreground" />
			<h3 class="text-sm font-medium text-foreground">Status History</h3>
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
				<p class="text-xs text-muted-foreground">No status data for this period</p>
			</div>
		{:else}
			<div class="h-[220px]">
				<canvas bind:this={canvasEl}></canvas>
			</div>
			<!-- Legend -->
			<div class="flex items-center justify-end space-x-4 mt-2">
				<div class="flex items-center space-x-1.5">
					<div class="w-2.5 h-2.5 rounded-sm bg-emerald-500/70"></div>
					<span class="text-[10px] text-muted-foreground">Running</span>
				</div>
				<div class="flex items-center space-x-1.5">
					<div class="w-2.5 h-2.5 rounded-sm bg-red-500/70"></div>
					<span class="text-[10px] text-muted-foreground">Down</span>
				</div>
			</div>
		{/if}
	</div>
</div>

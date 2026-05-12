<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2 } from 'lucide-svelte';
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

<section>
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-border pb-3">
		<div>
			<h3 class="text-sm font-medium text-foreground">Status History</h3>
			<p class="mt-0.5 text-xs text-muted-foreground">Each bar is one health check. Green = reachable, Red = down or unreachable.</p>
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
	<div class="pt-4">
		{#if loading}
			<div class="flex h-[220px] items-center justify-center">
				<Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
			</div>
		{:else if heartbeats.length === 0}
			<div class="flex h-[220px] items-center justify-center">
				<p class="text-xs text-muted-foreground">No status data for this period</p>
			</div>
		{:else}
			<div class="h-[220px]">
				<canvas bind:this={canvasEl}></canvas>
			</div>
			<!-- Legend -->
			<div class="mt-2 flex items-center justify-end gap-4 font-mono tabular-nums text-[10px] text-muted-foreground">
				<div class="flex items-center gap-1.5">
					<span class="inline-block h-1.5 w-1.5 rounded-full bg-success"></span>
					Running
				</div>
				<div class="flex items-center gap-1.5">
					<span class="inline-block h-1.5 w-1.5 rounded-full bg-destructive"></span>
					Down
				</div>
			</div>
		{/if}
	</div>
</section>

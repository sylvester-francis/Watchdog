<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2, Shield } from 'lucide-svelte';
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
		Tooltip
	} from 'chart.js';

	Chart.register(
		LineController,
		LineElement,
		PointElement,
		LinearScale,
		CategoryScale,
		Filler,
		Tooltip
	);

	interface Props {
		monitorId: string;
	}

	let { monitorId }: Props = $props();

	const toast = getToasts();

	type Period = '7d' | '30d' | '90d';
	const periods: { value: Period; label: string }[] = [
		{ value: '7d', label: '7D' },
		{ value: '30d', label: '30D' },
		{ value: '90d', label: '90D' }
	];

	let activePeriod = $state<Period>('30d');
	let heartbeats = $state<HeartbeatPoint[]>([]);
	let loading = $state(true);
	let canvasEl = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let chart: Chart | null = null;

	let latestExpiry = $derived.by(() => {
		const withExpiry = heartbeats.filter((h) => h.cert_expiry_days != null);
		if (withExpiry.length === 0) return null;
		return withExpiry[withExpiry.length - 1].cert_expiry_days!;
	});

	let expiryStatus = $derived.by(() => {
		if (latestExpiry === null) return { color: 'text-muted-foreground', label: 'Unknown' };
		if (latestExpiry < 7) return { color: 'text-red-400', label: 'Critical' };
		if (latestExpiry < 14) return { color: 'text-red-400', label: 'Expiring Soon' };
		if (latestExpiry < 30) return { color: 'text-amber-400', label: 'Renew Soon' };
		return { color: 'text-emerald-400', label: 'Healthy' };
	});

	function formatLabel(time: string, period: Period): string {
		const d = new Date(time);
		if (period === '7d') {
			return d.toLocaleDateString([], { weekday: 'short' });
		}
		return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
	}

	function expiryLineColor(days: number): string {
		if (days < 14) return '#ef4444';
		if (days < 30) return '#f59e0b';
		return '#22c55e';
	}

	function buildChart() {
		if (!canvasEl) return;

		if (chart) {
			chart.destroy();
			chart = null;
		}

		const withExpiry = heartbeats.filter((h) => h.cert_expiry_days != null);
		if (withExpiry.length === 0) return;

		const labels = withExpiry.map((p) => formatLabel(p.time, activePeriod));
		const expiryData = withExpiry.map((p) => p.cert_expiry_days!);

		// Color the line based on how close to expiry
		const currentDays = expiryData[expiryData.length - 1];
		const lineColor = expiryLineColor(currentDays);
		const fillColor = lineColor.replace(')', ', 0.08)').replace('rgb', 'rgba');

		chart = new Chart(canvasEl, {
			type: 'line',
			data: {
				labels,
				datasets: [
					{
						label: 'Days Until Expiry',
						data: expiryData,
						borderColor: lineColor,
						backgroundColor: fillColor,
						fill: true,
						tension: 0.2,
						borderWidth: 2,
						pointRadius: 1,
						pointHoverRadius: 4,
						pointBackgroundColor: lineColor,
						pointHoverBackgroundColor: lineColor
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
							label: (ctx: any) => ` ${ctx.parsed.y} days remaining`
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
							callback: (v: any) => `${v}d`
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
			toast.error(err instanceof Error ? err.message : 'Failed to load certificate data');
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
		<div>
			<div class="flex items-center space-x-2">
				<Shield class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">Certificate Expiry</h3>
				{#if latestExpiry !== null}
					<span class="text-xs font-mono font-medium {expiryStatus.color}">{latestExpiry}d — {expiryStatus.label}</span>
				{/if}
			</div>
			<p class="text-[10px] text-muted-foreground mt-0.5 ml-6">Days until certificate expires. A declining line is normal — watch for sudden drops after renewal.</p>
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
		{:else if heartbeats.filter(h => h.cert_expiry_days != null).length === 0}
			<div class="flex items-center justify-center h-[220px]">
				<p class="text-xs text-muted-foreground">No certificate expiry data for this period</p>
			</div>
		{:else}
			<div class="h-[220px]">
				<canvas bind:this={canvasEl}></canvas>
			</div>
		{/if}
	</div>
</div>

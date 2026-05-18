<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { TrendingUp, TrendingDown, ArrowRight, AlertCircle } from 'lucide-svelte';
	import { getLatencyTrend } from '$lib/api/monitors';
	import type { LatencyTrend, TrendWindow } from '$lib/api/monitors';
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

	let window = $state<TrendWindow>('7d');
	let trend = $state<LatencyTrend | null>(null);
	let loading = $state(true);
	let error = $state('');
	let canvas = $state<HTMLCanvasElement | null>(null);
	let chart: Chart | null = null;

	const windows: { value: TrendWindow; label: string }[] = [
		{ value: '7d', label: '7 days' },
		{ value: '30d', label: '30 days' },
		{ value: '90d', label: '90 days' }
	];

	async function load() {
		loading = true;
		error = '';
		try {
			trend = await getLatencyTrend(monitorId, window);
			renderChart();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load trend';
		} finally {
			loading = false;
		}
	}

	function renderChart() {
		if (!canvas || !trend) return;
		chart?.destroy();
		const labels = trend.points.map((p) => formatBucketLabel(p.time, window));
		chart = new Chart(canvas, {
			type: 'line',
			data: {
				labels,
				datasets: [
					{
						label: 'p50',
						data: trend.points.map((p) => p.p50),
						borderColor: 'rgb(34, 197, 94)',
						backgroundColor: 'transparent',
						borderWidth: 1.5,
						pointRadius: 0,
						tension: 0.3
					},
					{
						label: 'p95',
						data: trend.points.map((p) => p.p95),
						borderColor: 'rgb(59, 130, 246)',
						backgroundColor: 'transparent',
						borderWidth: 1.5,
						pointRadius: 0,
						tension: 0.3
					},
					{
						label: 'p99',
						data: trend.points.map((p) => p.p99),
						borderColor: 'rgb(239, 68, 68)',
						backgroundColor: 'transparent',
						borderWidth: 1.5,
						pointRadius: 0,
						tension: 0.3
					}
				]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				interaction: { intersect: false, mode: 'index' },
				plugins: {
					legend: { position: 'bottom', labels: { boxWidth: 12, font: { size: 11 } } },
					tooltip: {
						callbacks: { label: (ctx) => `${ctx.dataset.label}: ${ctx.formattedValue} ms` }
					}
				},
				scales: {
					x: { grid: { display: false }, ticks: { maxTicksLimit: 8, font: { size: 10 } } },
					y: {
						beginAtZero: true,
						ticks: { callback: (v) => `${v} ms`, font: { size: 10 } },
						grid: { color: 'rgba(255,255,255,0.05)' }
					}
				}
			}
		});
	}

	function formatBucketLabel(iso: string, w: TrendWindow): string {
		const d = new Date(iso);
		if (w === '7d') return d.toLocaleString([], { weekday: 'short', hour: '2-digit' });
		return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
	}

	function pctChange(current: number, previous: number): number | null {
		if (previous === 0) return null;
		return Math.round(((current - previous) / previous) * 100);
	}

	function deltaText(pct: number | null): { label: string; direction: 'up' | 'down' | 'flat' } {
		if (pct === null) return { label: 'no prior data', direction: 'flat' };
		if (pct === 0) return { label: 'no change vs last period', direction: 'flat' };
		if (pct > 0) return { label: `+${pct}% vs last period — slower`, direction: 'up' };
		return { label: `${pct}% vs last period — faster`, direction: 'down' };
	}

	async function setWindow(w: TrendWindow) {
		if (w === window) return;
		window = w;
		await load();
	}

	let p95Delta = $derived(trend ? pctChange(trend.current.p95, trend.previous.p95) : null);
	let p99Delta = $derived(trend ? pctChange(trend.current.p99, trend.previous.p99) : null);
	let p50Delta = $derived(trend ? pctChange(trend.current.p50, trend.previous.p50) : null);
	let headlineDelta = $derived(deltaText(p95Delta));

	onMount(load);
	onDestroy(() => chart?.destroy());
</script>

<section class="bg-card border border-border/50 rounded-lg">
	<div class="px-5 py-3 border-b border-border/50 flex items-center justify-between gap-3">
		<div>
			<h2 class="text-sm font-semibold text-foreground">Latency trend</h2>
			<p class="text-[11px] text-muted-foreground mt-0.5">
				p50/p95/p99 across successful checks
			</p>
		</div>
		<div class="flex items-center gap-1 text-xs">
			{#each windows as w (w.value)}
				<button
					onclick={() => setWindow(w.value)}
					class="px-2 py-1 rounded border border-border/50 transition
						{window === w.value
							? 'bg-foreground text-background border-foreground'
							: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
				>
					{w.label}
				</button>
			{/each}
		</div>
	</div>

	{#if loading && !trend}
		<div class="p-8 text-center text-xs text-muted-foreground">Loading…</div>
	{:else if error}
		<div class="p-5 text-xs text-destructive flex items-center gap-1">
			<AlertCircle class="w-3 h-3" />{error}
		</div>
	{:else if trend && trend.current.sample_count === 0}
		<div class="p-8 text-center">
			<p class="text-sm text-muted-foreground">No checks in the last {window}.</p>
			<p class="text-xs text-muted-foreground/70 mt-1">Try a wider window or wait for data.</p>
		</div>
	{:else if trend}
		<div class="px-5 py-4 border-b border-border/50">
			<div class="flex items-baseline gap-3 mb-3">
				<div class="text-2xl font-semibold text-foreground tabular-nums">
					{trend.current.p95}<span class="text-base text-muted-foreground"> ms</span>
				</div>
				<div class="text-xs font-medium flex items-center gap-1
					{headlineDelta.direction === 'up'
						? 'text-warning'
						: headlineDelta.direction === 'down'
							? 'text-success'
							: 'text-muted-foreground'}">
					{#if headlineDelta.direction === 'up'}
						<TrendingUp class="w-3.5 h-3.5" />
					{:else if headlineDelta.direction === 'down'}
						<TrendingDown class="w-3.5 h-3.5" />
					{:else}
						<ArrowRight class="w-3.5 h-3.5" />
					{/if}
					p95 {headlineDelta.label}
				</div>
			</div>
			<div class="grid grid-cols-3 gap-px bg-border/50 text-xs">
				<div class="bg-card p-2">
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground">p50 (median)</div>
					<div class="text-foreground tabular-nums mt-0.5">
						{trend.current.p50}<span class="text-muted-foreground"> ms</span>
					</div>
					<div class="text-[10px] text-muted-foreground tabular-nums">
						{p50Delta === null ? '—' : (p50Delta > 0 ? '+' : '') + p50Delta + '%'}
					</div>
				</div>
				<div class="bg-card p-2">
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground">p95</div>
					<div class="text-foreground tabular-nums mt-0.5">
						{trend.current.p95}<span class="text-muted-foreground"> ms</span>
					</div>
					<div class="text-[10px] text-muted-foreground tabular-nums">
						{p95Delta === null ? '—' : (p95Delta > 0 ? '+' : '') + p95Delta + '%'}
					</div>
				</div>
				<div class="bg-card p-2">
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground">p99 (tail)</div>
					<div class="text-foreground tabular-nums mt-0.5">
						{trend.current.p99}<span class="text-muted-foreground"> ms</span>
					</div>
					<div class="text-[10px] text-muted-foreground tabular-nums">
						{p99Delta === null ? '—' : (p99Delta > 0 ? '+' : '') + p99Delta + '%'}
					</div>
				</div>
			</div>
		</div>
		<div class="p-4">
			<div class="relative h-64">
				<canvas bind:this={canvas}></canvas>
			</div>
			<p class="text-[11px] text-muted-foreground mt-2">
				{trend.current.sample_count.toLocaleString()} checks over {window} · bucket: {trend.bucket_interval}
			</p>
		</div>
	{/if}
</section>

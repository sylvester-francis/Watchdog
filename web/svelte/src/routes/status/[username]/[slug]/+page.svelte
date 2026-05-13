<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { Shield, CheckCircle2, Bell } from 'lucide-svelte';
	import { statusPages as statusPagesApi } from '$lib/api';
	import type { PublicStatusPageData, PublicMonitorData } from '$lib/types';

	let data = $state<PublicStatusPageData | null>(null);
	let loading = $state(true);
	let error = $state('');
	let refreshTimer: ReturnType<typeof setInterval>;

	let username = $derived((page.params as Record<string, string>).username?.replace(/^@/, '') ?? '');
	let slug = $derived((page.params as Record<string, string>).slug ?? '');

	function statusPipClass(status: string): string {
		if (status === 'up') return 'bg-success';
		if (status === 'down') return 'bg-destructive';
		return 'bg-muted-foreground/50';
	}

	function statusTextClass(status: string): string {
		if (status === 'up') return 'text-success';
		if (status === 'down') return 'text-destructive';
		return 'text-muted-foreground';
	}

	function statusLabel(m: PublicMonitorData): string {
		if (m.status === 'up') {
			if (m.type === 'docker') return 'Running';
			if (m.type === 'system') return 'Healthy';
			return 'Operational';
		}
		if (m.status === 'down') {
			if (m.type === 'docker') return 'Stopped';
			if (m.type === 'system') return 'Threshold Exceeded';
			return 'Down';
		}
		return 'Unknown';
	}

	function uptimeBarClass(percent: number): string {
		if (percent < 0) return 'h-6 bg-muted/30';
		if (percent >= 99) return 'h-6 bg-success/70';
		if (percent >= 50) return 'h-6 bg-warning/70';
		return 'h-6 bg-destructive/70';
	}

	function uptimePercentClass(percent: number): string {
		if (percent < 0) return 'text-muted-foreground';
		if (percent >= 99) return 'text-success';
		if (percent >= 95) return 'text-warning';
		return 'text-destructive';
	}

	function uptimeBarTitle(day: { date: string; percent: number }): string {
		if (day.percent < 0) return `${day.date}: No data`;
		return `${day.date}: ${Math.round(day.percent)}% uptime`;
	}

	function formatPercent(v: number): string {
		if (v < 0) return '—';
		return v.toFixed(1);
	}

	function formatTimeAgo(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		const days = Math.floor(hrs / 24);
		return `${days}d ago`;
	}

	function formatDuration(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		const mins = Math.floor(seconds / 60);
		if (mins < 60) return `${mins}m`;
		const hrs = Math.floor(mins / 60);
		const remainMins = mins % 60;
		return remainMins > 0 ? `${hrs}h ${remainMins}m` : `${hrs}h`;
	}

	function monitoringSinceLabel(m: PublicMonitorData): string {
		if (m.data_days < 90) {
			const d = new Date(m.monitoring_since);
			return `Since ${d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`;
		}
		return '90 days ago';
	}

	function incidentStatusClass(status: string): string {
		if (status === 'open') return 'text-destructive';
		if (status === 'acknowledged') return 'text-warning';
		return 'text-success';
	}

	async function loadData() {
		try {
			data = await statusPagesApi.getPublicStatusPage(username, slug);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Status page not found';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadData();
		refreshTimer = setInterval(loadData, 60000);
	});

	onDestroy(() => {
		clearInterval(refreshTimer);
	});
</script>

<svelte:head>
	<title>{data?.page?.name ?? 'Status'} — Status</title>
</svelte:head>

<div class="min-h-screen bg-background font-sans text-foreground antialiased">
	<!-- Nav -->
	<nav class="border-b border-border bg-background/80 backdrop-blur-sm">
		<div class="mx-auto flex h-14 max-w-3xl items-center justify-between px-4 sm:px-6">
			<a href="/" class="flex items-center gap-2.5">
				<div class="flex h-7 w-7 items-center justify-center rounded-md bg-accent">
					<Shield class="h-3.5 w-3.5 text-white" />
				</div>
				<span class="text-sm font-semibold text-foreground">WatchDog</span>
			</a>
			<div class="flex items-center gap-4 text-xs">
				<a href="#subscribe" class="text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline">
					Subscribe
				</a>
				<a href="/register" class="bg-accent px-3 py-1.5 font-medium text-background transition-opacity hover:opacity-90">
					Sign Up
				</a>
			</div>
		</div>
	</nav>

	{#if loading}
		<div class="mx-auto max-w-3xl px-4 py-10 sm:px-6 sm:py-14">
			<div class="mb-10 text-center">
				<div class="mx-auto h-7 w-48 animate-pulse bg-muted/50"></div>
				<div class="mx-auto mt-3 h-4 w-64 animate-pulse bg-muted/30"></div>
			</div>
			<div class="mb-10 h-16 w-full animate-pulse bg-muted/30"></div>
			<div class="space-y-2">
				{#each Array(3) as _}
					<div class="h-10 animate-pulse bg-muted/30"></div>
				{/each}
			</div>
		</div>
	{:else if error}
		<div class="mx-auto max-w-3xl px-4 py-20 sm:px-6">
			<p class="text-center text-sm text-muted-foreground">{error}</p>
		</div>
	{:else if data}
		<div class="mx-auto max-w-3xl px-4 py-10 sm:px-6 sm:py-14">
			<!-- Header -->
			<header class="text-center">
				<h1 class="text-xl font-medium tracking-tight text-foreground sm:text-2xl md:text-3xl">{data.page.name}</h1>
				{#if data.page.description}
					<p class="mx-auto mt-2 max-w-md text-sm text-muted-foreground">{data.page.description}</p>
				{/if}
			</header>

			<!-- Overall Status Banner — hairline section -->
			<section class="mt-10">
				<div class="flex items-center justify-center gap-2 font-mono tabular-nums">
					{#if data.monitors.length > 0}
						<span class="inline-block h-1.5 w-1.5 rounded-full {data.all_up ? 'bg-success' : 'bg-destructive'} {data.all_up ? '' : 'animate-pulse'}"></span>
					{:else}
						<span class="inline-block h-1.5 w-1.5 rounded-full bg-muted-foreground/50"></span>
					{/if}
					<span class="text-sm uppercase tracking-wider {data.all_up && data.monitors.length > 0 ? 'text-success' : data.monitors.length > 0 ? 'text-destructive' : 'text-muted-foreground'}">
						{data.overall_status === 'operational' ? 'All Systems Operational' : data.overall_status === 'degraded' ? 'Some Systems Experiencing Issues' : 'No Monitors Configured'}
					</span>
				</div>
				{#if data.aggregate_uptime > 0 && data.monitors.length > 0}
					<p class="mt-2 text-center font-mono tabular-nums text-xs text-muted-foreground">
						{formatPercent(data.aggregate_uptime)}% uptime over the last 90 days
					</p>
				{/if}
			</section>

			<!-- Monitors -->
			{#if data.monitors.length > 0}
				<section class="mt-10">
					<div class="border-b border-border pb-3">
						<h2 class="text-sm font-medium text-foreground">Services</h2>
					</div>
					<div class="divide-y divide-border/40">
						{#each data.monitors as m}
							<div class="py-4">
								<div class="flex items-center justify-between gap-3">
									<div class="flex min-w-0 items-center gap-2">
										<span class="inline-block h-1.5 w-1.5 shrink-0 rounded-full {statusPipClass(m.status)}"></span>
										<span class="truncate text-sm font-medium text-foreground">{m.name}</span>
										<span class="hidden font-mono tabular-nums text-[10px] uppercase tracking-wider text-muted-foreground sm:inline">{m.type}</span>
									</div>
									<div class="flex shrink-0 flex-col items-end gap-0.5 font-mono tabular-nums text-xs sm:flex-row sm:items-center sm:gap-4">
										{#if m.metric_value}
											<span class="text-muted-foreground">{m.metric_value}</span>
										{:else if m.has_latency}
											<span class="text-muted-foreground">{m.latency_ms}ms</span>
										{/if}
										{#if m.uptime_percent >= 0}
											<span class="font-medium {uptimePercentClass(m.uptime_percent)}">{formatPercent(m.uptime_percent)}%</span>
										{/if}
										<span class="uppercase tracking-wider {statusTextClass(m.status)}">
											{statusLabel(m)}
										</span>
									</div>
								</div>
								{#if m.uptime_history && m.uptime_history.length > 0}
									<div class="mt-3">
										<div class="flex items-end gap-px">
											{#each m.uptime_history as day}
												<div class="flex-1 {uptimeBarClass(day.percent)}" title={uptimeBarTitle(day)}></div>
											{/each}
										</div>
										<div class="mt-1.5 flex items-center justify-between font-mono tabular-nums text-[10px] text-muted-foreground/60">
											<span>{monitoringSinceLabel(m)}</span>
											<span>Today</span>
										</div>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				</section>
			{:else}
				<section class="mt-10">
					<p class="text-center text-sm text-muted-foreground">No monitors configured for this status page.</p>
				</section>
			{/if}

			<!-- Incident History -->
			<section class="mt-10">
				<div class="border-b border-border pb-3">
					<h2 class="text-sm font-medium text-foreground">Incident History</h2>
				</div>
				{#if data.incidents.length > 0}
					<div class="divide-y divide-border/40">
						{#each data.incidents as inc}
							<div class="flex items-start justify-between gap-3 py-3 sm:items-center">
								<div class="min-w-0">
									<div class="flex items-baseline gap-2">
										<span class="font-mono tabular-nums text-[11px] uppercase tracking-wider {incidentStatusClass(inc.status)}">
											{inc.status}
										</span>
										<span class="truncate text-sm font-medium text-foreground">{inc.monitor_name}</span>
									</div>
									<p class="mt-1 font-mono tabular-nums text-[11px] text-muted-foreground">
										{formatTimeAgo(inc.started_at)}{#if !inc.is_active} · Resolved in {formatDuration(inc.duration_seconds)}{/if}
									</p>
								</div>
								{#if inc.is_active}
									<span class="mt-1.5 inline-block h-1.5 w-1.5 shrink-0 animate-pulse rounded-full bg-destructive sm:mt-0"></span>
								{/if}
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex items-center justify-center gap-2 pt-4 font-mono tabular-nums text-sm text-success">
						<CheckCircle2 class="h-4 w-4" />
						<span>No incidents in the last 30 days.</span>
					</div>
				{/if}
			</section>

			<!-- Subscribe Section -->
			<section id="subscribe" class="mt-10">
				<div class="border-b border-border pb-3">
					<h2 class="text-sm font-medium text-foreground">Subscribe to Updates</h2>
				</div>
				<div class="flex flex-col items-start gap-3 pt-4 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
					<p class="text-xs text-muted-foreground">
						<Bell class="mr-1 inline-block h-3 w-3" />
						Get notified when something goes wrong.
					</p>
					<button disabled class="cursor-not-allowed border border-border bg-muted/30 px-3 py-1.5 text-xs font-medium text-muted-foreground">
						Coming Soon
					</button>
				</div>
			</section>

			<!-- Footer -->
			<footer class="mt-12 border-t border-border pt-6">
				<div class="flex flex-col items-center justify-between gap-4 font-mono tabular-nums text-[11px] text-muted-foreground/60 sm:flex-row">
					<a href="/" class="inline-flex items-center gap-1.5 transition-colors hover:text-muted-foreground">
						<Shield class="h-3 w-3" />
						<span>Powered by WatchDog</span>
					</a>
					<div class="flex items-center gap-4">
						<a href="/register" class="transition-colors hover:text-muted-foreground">
							Create your own status page
						</a>
					</div>
				</div>
			</footer>
		</div>
	{/if}
</div>

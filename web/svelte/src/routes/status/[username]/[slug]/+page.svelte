<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { ShieldCheck, Activity, CheckCircle2, Bell } from 'lucide-svelte';
	import { statusPages as statusPagesApi } from '$lib/api';
	import type { PublicStatusPageData, PublicMonitorData, PublicIncidentData } from '$lib/types';

	let data = $state<PublicStatusPageData | null>(null);
	let loading = $state(true);
	let error = $state('');
	let refreshTimer: ReturnType<typeof setInterval>;

	let username = $derived((page.params as Record<string, string>).username?.replace(/^@/, '') ?? '');
	let slug = $derived((page.params as Record<string, string>).slug ?? '');

	function typeBadgeClass(type: string): string {
		const map: Record<string, string> = {
			http: 'bg-blue-500/10 text-blue-400',
			tcp: 'bg-purple-500/10 text-purple-400',
			ping: 'bg-amber-500/10 text-amber-400',
			dns: 'bg-cyan-500/10 text-cyan-400',
			tls: 'bg-emerald-500/10 text-emerald-400',
			docker: 'bg-indigo-500/10 text-indigo-400',
			database: 'bg-sky-500/10 text-sky-400',
			system: 'bg-orange-500/10 text-orange-400',
		};
		return map[type] ?? 'bg-muted text-muted-foreground';
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
		if (percent < 0) return 'h-4 bg-muted/10 border border-dashed border-muted/20';
		if (percent >= 99) return 'h-8 bg-emerald-500/70';
		if (percent >= 50) return 'h-8 bg-amber-500/70';
		return 'h-8 bg-red-500/70';
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
			return `Monitoring since ${d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`;
		}
		return '90 days ago';
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
		// Auto-refresh every 60 seconds
		refreshTimer = setInterval(loadData, 60000);
	});

	onDestroy(() => {
		clearInterval(refreshTimer);
	});
</script>

<svelte:head>
	<title>{data?.page?.name ?? 'Status'} — Status</title>
</svelte:head>

<div class="bg-background text-foreground font-sans min-h-screen antialiased">
	<!-- Nav -->
	<nav class="border-b border-border bg-background/80 backdrop-blur-sm">
		<div class="max-w-3xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
			<div class="flex items-center space-x-3">
				<a href="/" class="flex items-center space-x-2.5">
					<div class="w-7 h-7 bg-accent rounded-md flex items-center justify-center">
						<ShieldCheck class="w-3.5 h-3.5 text-white" />
					</div>
					<span class="text-sm font-semibold text-foreground">WatchDog</span>
				</a>
				<span class="px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider rounded bg-amber-500/15 text-amber-400 border border-amber-500/20">Beta</span>
			</div>
			<div class="flex items-center space-x-2">
				<a href="#subscribe" class="px-3 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground border border-border rounded-md transition-colors">
					Subscribe
				</a>
				<a href="/register" class="px-3.5 py-1.5 bg-accent text-accent-foreground hover:bg-accent/90 text-xs font-medium rounded-md transition-colors">
					Sign Up
				</a>
			</div>
		</div>
	</nav>

	{#if loading}
		<div class="max-w-3xl mx-auto px-4 sm:px-6 py-10 sm:py-14">
			<div class="text-center mb-10">
				<div class="h-7 w-48 bg-muted/50 rounded animate-pulse mx-auto"></div>
				<div class="h-4 w-64 bg-muted/30 rounded animate-pulse mx-auto mt-3"></div>
			</div>
			<div class="h-16 w-full bg-muted/30 rounded-lg animate-pulse mb-10"></div>
			<div class="bg-card border border-border rounded-lg">
				{#each Array(3) as _}
					<div class="px-5 py-4 border-b border-border/20">
						<div class="h-4 w-48 bg-muted/50 rounded animate-pulse"></div>
					</div>
				{/each}
			</div>
		</div>
	{:else if error}
		<div class="max-w-3xl mx-auto px-4 sm:px-6 py-20 text-center">
			<Activity class="w-8 h-8 text-muted-foreground/30 mx-auto mb-3" />
			<p class="text-sm text-muted-foreground">{error}</p>
		</div>
	{:else if data}
		<div class="max-w-3xl mx-auto px-4 sm:px-6 py-10 sm:py-14">

			<!-- Header -->
			<div class="text-center mb-10">
				<h1 class="text-2xl font-bold text-foreground tracking-tight">{data.page.name}</h1>
				{#if data.page.description}
					<p class="text-sm text-muted-foreground mt-2 max-w-md mx-auto">{data.page.description}</p>
				{/if}
			</div>

			<!-- Overall Status Banner -->
			<div class="rounded-lg border p-5 mb-10 text-center {data.all_up && data.monitors.length > 0 ? 'bg-emerald-500/[0.07] border-emerald-500/20' : data.monitors.length > 0 ? 'bg-red-500/[0.07] border-red-500/20' : 'bg-card border-border'}">
				<div class="flex items-center justify-center space-x-2.5">
					{#if data.monitors.length > 0}
						<div class="w-2 h-2 rounded-full animate-pulse-dot {data.all_up ? 'bg-emerald-400' : 'bg-red-400'}"></div>
					{/if}
					<span class="text-sm font-medium {data.all_up && data.monitors.length > 0 ? 'text-emerald-400' : data.monitors.length > 0 ? 'text-red-400' : 'text-muted-foreground'}">
						{data.overall_status === 'operational' ? 'All Systems Operational' : data.overall_status === 'degraded' ? 'Some Systems Experiencing Issues' : 'No Monitors Configured'}
					</span>
				</div>
				{#if data.aggregate_uptime > 0 && data.monitors.length > 0}
					<p class="text-xs text-muted-foreground mt-1.5">{formatPercent(data.aggregate_uptime)}% uptime over the last 90 days</p>
				{/if}
			</div>

			<!-- Monitors -->
			{#if data.monitors.length > 0}
				<div class="bg-card border border-border rounded-lg overflow-hidden">
					{#each data.monitors as m, i}
						<div class="{i > 0 ? 'border-t border-border' : ''}">
							<div class="px-3 sm:px-5 py-3.5 flex items-center justify-between">
								<div class="flex items-center space-x-2 sm:space-x-3 min-w-0">
									<div class="w-2 h-2 rounded-full shrink-0 {m.status === 'up' ? 'bg-emerald-400' : m.status === 'down' ? 'bg-red-400' : 'bg-zinc-500'}"></div>
									<span class="text-sm font-medium text-foreground truncate">{m.name}</span>
									<span class="hidden sm:inline-flex px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider rounded {typeBadgeClass(m.type)}">{m.type}</span>
								</div>
								<div class="flex items-center space-x-3 sm:space-x-4 shrink-0 ml-2 sm:ml-4">
									{#if m.metric_value}
										<span class="text-xs text-muted-foreground tabular-nums">{m.metric_value}</span>
									{:else if m.has_latency}
										<span class="text-xs text-muted-foreground tabular-nums">{m.latency_ms}ms</span>
									{/if}
									{#if m.uptime_percent >= 0}
										<span class="text-xs font-medium tabular-nums {m.uptime_percent >= 99 ? 'text-emerald-400' : m.uptime_percent >= 95 ? 'text-amber-400' : 'text-red-400'}">{formatPercent(m.uptime_percent)}%</span>
									{/if}
									<span class="text-xs font-medium {m.status === 'up' ? 'text-emerald-400' : m.status === 'down' ? 'text-red-400' : 'text-muted-foreground'}">
										{statusLabel(m)}
									</span>
								</div>
							</div>
							{#if m.uptime_history && m.uptime_history.length > 0}
								<div class="px-3 sm:px-5 pb-3">
									<div class="flex items-end space-x-px">
										{#each m.uptime_history as day}
											<div class="flex-1 rounded-sm {uptimeBarClass(day.percent)}" title={uptimeBarTitle(day)}></div>
										{/each}
									</div>
									<div class="flex items-center justify-between mt-1.5">
										<span class="text-[9px] text-muted-foreground/50">{monitoringSinceLabel(m)}</span>
										<span class="text-[9px] text-muted-foreground/50">Today</span>
									</div>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{:else}
				<div class="bg-card border border-border rounded-lg p-10 text-center">
					<Activity class="w-8 h-8 text-muted-foreground/30 mx-auto mb-3" />
					<p class="text-sm text-muted-foreground">No monitors configured for this status page.</p>
				</div>
			{/if}

			<!-- Incident History -->
			<div class="mt-10">
				<h2 class="text-sm font-semibold text-foreground mb-4">Incident History</h2>
				{#if data.incidents.length > 0}
					<div class="bg-card border border-border rounded-lg overflow-hidden divide-y divide-border">
						{#each data.incidents as inc}
							<div class="px-3 sm:px-5 py-3.5 flex items-start sm:items-center justify-between gap-3">
								<div class="min-w-0">
									<div class="flex items-center space-x-2">
										<span class="px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider rounded {inc.status === 'open' ? 'bg-red-500/10 text-red-400' : inc.status === 'acknowledged' ? 'bg-amber-500/10 text-amber-400' : 'bg-emerald-500/10 text-emerald-400'}">{inc.status}</span>
										<span class="text-sm font-medium text-foreground truncate">{inc.monitor_name}</span>
									</div>
									<p class="text-xs text-muted-foreground mt-1">
										{formatTimeAgo(inc.started_at)}
										{#if !inc.is_active}
											&middot; Resolved in {formatDuration(inc.duration_seconds)}
										{/if}
									</p>
								</div>
								{#if inc.is_active}
									<div class="w-2 h-2 rounded-full bg-red-400 animate-pulse-dot shrink-0 mt-1.5 sm:mt-0"></div>
								{/if}
							</div>
						{/each}
					</div>
				{:else}
					<div class="bg-card border border-border rounded-lg p-8 text-center">
						<CheckCircle2 class="w-6 h-6 text-emerald-400/60 mx-auto mb-2" />
						<p class="text-sm text-muted-foreground">No incidents in the last 30 days.</p>
					</div>
				{/if}
			</div>

			<!-- Subscribe Section -->
			<div id="subscribe" class="mt-10">
				<div class="bg-card border border-border rounded-lg p-6 text-center">
					<Bell class="w-6 h-6 text-muted-foreground/40 mx-auto mb-2.5" />
					<h3 class="text-sm font-medium text-foreground mb-1">Subscribe to Updates</h3>
					<p class="text-xs text-muted-foreground mb-4">Get notified when something goes wrong.</p>
					<button disabled class="px-4 py-2 bg-muted/50 text-muted-foreground text-xs font-medium rounded-md cursor-not-allowed">
						Coming Soon
					</button>
				</div>
			</div>

			<!-- Footer -->
			<div class="mt-12 pt-6 border-t border-border">
				<div class="flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-muted-foreground/50">
					<div class="flex items-center space-x-2">
						<a href="/" class="inline-flex items-center space-x-1.5 hover:text-muted-foreground transition-colors">
							<ShieldCheck class="w-3 h-3" />
							<span>Powered by WatchDog</span>
						</a>
						<span class="px-1 py-0.5 text-[9px] font-semibold uppercase tracking-wider rounded bg-amber-500/15 text-amber-400/70 border border-amber-500/10">Beta</span>
					</div>
					<div class="flex items-center space-x-4">
						<a href="https://github.com/sylvester-francis/watchdog" target="_blank" rel="noopener noreferrer" class="hover:text-muted-foreground transition-colors">GitHub</a>
						<a href="/register" class="hover:text-muted-foreground transition-colors">Create your own status page</a>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>

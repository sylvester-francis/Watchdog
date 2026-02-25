<script lang="ts">
	import { ChevronRight, ArrowRight } from 'lucide-svelte';
	import { formatPercent, uptimeColor, isInfraMonitor } from '$lib/utils';
	import type { MonitorSummary } from '$lib/types';
	import UptimeChecks from './UptimeChecks.svelte';
	import Sparkline from './Sparkline.svelte';

	interface Props {
		monitors: MonitorSummary[];
		title: string;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon: any;
		variant: 'service' | 'infra';
	}

	let { monitors, title, icon, variant }: Props = $props();

	function uptimePercent(m: MonitorSummary): number {
		if (m.total === 0) return 0;
		return (m.uptimeUp / m.total) * 100;
	}

	function sparkColor(status: string): string {
		if (status === 'up') return '#22c55e';
		if (status === 'down') return '#ef4444';
		return '#a1a1aa';
	}

	function lastLatency(m: MonitorSummary): string {
		if (!m.latencies || m.latencies.length === 0) {
			return m.status === 'up' ? 'OK' : '--';
		}
		return m.latencies[m.latencies.length - 1] + 'ms';
	}

	function checkResults(m: MonitorSummary): number[] {
		if (m.total === 0) return [];
		// Build check results from the last heartbeats
		// uptimeUp + uptimeDown = total; but we don't have per-check granularity from this endpoint
		// Use latencies length as an approximation: if latency exists = up, gaps = down
		const results: number[] = [];
		const up = m.uptimeUp;
		const down = m.uptimeDown;
		// Interleave: all ups first then downs (simplified)
		for (let i = 0; i < up && results.length < 20; i++) results.push(1);
		for (let i = 0; i < down && results.length < 20; i++) results.push(0);
		return results;
	}

	function parseMetricValue(msg: string | undefined): string | null {
		if (!msg) return null;
		const match = msg.match(/([\d.]+)%/);
		return match ? `${parseFloat(match[1]).toFixed(1)}%` : null;
	}

	function infraValue(m: MonitorSummary): string {
		if (m.type === 'docker') return m.status === 'up' ? 'Running' : 'Stopped';
		if (m.type === 'service') return m.status === 'up' ? 'Running' : 'Stopped';
		if (m.type === 'database' && m.latencies?.length > 0) return m.latencies[m.latencies.length - 1] + 'ms';
		if (m.type === 'system') {
			const val = parseMetricValue(m.latest_value);
			if (val) return val;
			return m.status === 'up' ? 'OK' : 'Error';
		}
		if (m.latencies?.length > 0) return m.latencies[m.latencies.length - 1] + 'ms';
		return m.status === 'up' ? 'OK' : 'No data';
	}

	function infraValueClass(m: MonitorSummary): string {
		if (m.type === 'docker' || m.type === 'service') return m.status === 'up' ? 'text-emerald-400' : 'text-red-400';
		if (m.type === 'system') return m.status === 'up' ? 'text-emerald-400' : 'text-red-400';
		return 'text-muted-foreground';
	}
</script>

<div class="bg-card border border-border rounded-lg mb-4">
	<div class="px-4 py-3 border-b border-border flex items-center justify-between">
		<div class="flex items-center space-x-2">
			<icon class="w-4 h-4 text-muted-foreground"></icon>
			<h2 class="text-sm font-medium text-foreground">{title}</h2>
			<span class="text-[10px] text-muted-foreground font-mono">{monitors.length}</span>
		</div>
		<a href="/monitors" class="text-xs text-muted-foreground hover:text-foreground transition-colors flex items-center space-x-1">
			<span>View all</span>
			<ArrowRight class="w-3 h-3" />
		</a>
	</div>

	<!-- Column Headers -->
	{#if variant === 'service'}
		<div class="hidden sm:flex items-center px-4 py-2 border-b border-border/30 text-[9px] font-medium text-muted-foreground uppercase tracking-wider">
			<div class="w-5 shrink-0"></div>
			<div class="flex-1 min-w-0 ml-2">Service</div>
			<div class="w-36 shrink-0 text-center hidden md:block">Uptime (24h)</div>
			<div class="w-14 shrink-0 text-right hidden md:block ml-2">Uptime</div>
			<div class="w-16 shrink-0 text-right hidden md:block ml-3">Latency</div>
			<div class="w-14 shrink-0 text-right hidden md:block ml-2">Response</div>
			<div class="w-5 shrink-0"></div>
		</div>
	{:else}
		<div class="hidden sm:flex items-center px-4 py-2 border-b border-border/30 text-[9px] font-medium text-muted-foreground uppercase tracking-wider">
			<div class="w-5 shrink-0"></div>
			<div class="flex-1 min-w-0 ml-2">Service</div>
			<div class="w-36 shrink-0 text-center hidden md:block">Health (24h)</div>
			<div class="w-14 shrink-0 text-right hidden md:block ml-2">Uptime</div>
			<div class="w-20 shrink-0 text-right hidden md:block ml-3">Value</div>
			<div class="w-5 shrink-0"></div>
		</div>
	{/if}

	<!-- Rows -->
	<div class="divide-y divide-border/20">
		{#each monitors as m (m.id)}
			<a href="/monitors/{m.id}" class="flex items-center px-4 py-3.5 hover:bg-card-elevated transition-colors group">
				<div class="w-5 shrink-0 flex justify-center">
					<div class="w-2 h-2 rounded-full
						{m.status === 'up' ? 'bg-emerald-400 animate-pulse' : m.status === 'down' ? 'bg-red-400' : 'bg-muted-foreground'}"
						aria-label="Status: {m.status}"></div>
				</div>
				<div class="flex-1 min-w-0 ml-2">
					<div class="flex items-center space-x-2">
						<span class="text-sm text-foreground truncate">{m.name}</span>
						<span class="text-[9px] text-muted-foreground font-mono uppercase shrink-0 px-1.5 py-0.5 rounded bg-muted/50">{m.type}</span>
					</div>
					<p class="text-[10px] text-muted-foreground font-mono truncate mt-0.5 hidden sm:block">{m.target}</p>
				</div>

				<!-- Uptime checks -->
				<div class="w-36 shrink-0 hidden md:flex items-center justify-center mx-2">
					{#if m.total > 0}
						<UptimeChecks checkResults={checkResults(m)} />
					{:else}
						<span class="text-[9px] text-muted-foreground">No data</span>
					{/if}
				</div>

				<!-- Uptime % -->
				<div class="w-14 shrink-0 hidden md:flex items-center justify-end ml-2">
					<span class="text-xs font-mono font-medium {uptimeColor(uptimePercent(m))}">{formatPercent(uptimePercent(m))}%</span>
				</div>

				{#if variant === 'service'}
					<!-- Sparkline -->
					<div class="w-16 shrink-0 hidden md:flex items-center justify-end ml-3">
						{#if m.latencies?.length > 0}
							<Sparkline values={m.latencies} color={sparkColor(m.status)} />
						{:else}
							<span class="text-[9px] text-muted-foreground">No data</span>
						{/if}
					</div>
					<!-- Response time -->
					<div class="w-14 shrink-0 hidden md:flex items-center justify-end ml-2">
						<span class="text-xs font-mono text-muted-foreground">{lastLatency(m)}</span>
					</div>
				{:else}
					<!-- Value column for infra -->
					<div class="w-20 shrink-0 hidden md:flex items-center justify-end ml-3">
						<span class="text-xs font-mono {infraValueClass(m)}">{infraValue(m)}</span>
					</div>
				{/if}

				<div class="w-5 shrink-0 flex justify-end">
					<ChevronRight class="w-3 h-3 text-muted-foreground/20 group-hover:text-muted-foreground transition-colors" />
				</div>
			</a>
		{/each}
	</div>
</div>

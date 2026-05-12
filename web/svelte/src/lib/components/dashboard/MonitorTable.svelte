<script lang="ts">
	import { ChevronRight, ArrowRight } from 'lucide-svelte';
	import { formatPercent, uptimeColor } from '$lib/utils';
	import type { MonitorSummary } from '$lib/types';
	import UptimeChecks from './UptimeChecks.svelte';
	import { Sparkline } from '@sylvester-francis/watchdog-ui';
	import { StatusDot } from '@sylvester-francis/watchdog-ui';
	import { Pill } from '@sylvester-francis/watchdog-ui';

	interface Props {
		monitors: MonitorSummary[];
		title: string;
		variant: 'service' | 'infra';
	}

	let { monitors, title, variant }: Props = $props();

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

<section>
	<div class="flex items-center justify-between border-b border-border pb-3">
		<div class="flex items-baseline gap-2">
			<h3 class="text-sm font-medium text-foreground">{title}</h3>
			<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{monitors.length}</span>
		</div>
		<a href="/monitors" class="flex items-center gap-1 text-xs text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline">
			<span>View all</span>
			<ArrowRight class="h-3 w-3" />
		</a>
	</div>

	<!-- Column Headers -->
	{#if variant === 'service'}
		<div class="mt-4 hidden items-center pb-2 text-[9px] font-medium uppercase tracking-wider text-muted-foreground sm:flex">
			<div class="w-5 shrink-0"></div>
			<div class="ml-2 min-w-0 flex-1">Service</div>
			<div class="hidden w-36 shrink-0 text-center md:block">Uptime (24h)</div>
			<div class="ml-2 hidden w-14 shrink-0 text-right md:block">Uptime</div>
			<div class="ml-3 hidden w-16 shrink-0 text-right md:block">Latency</div>
			<div class="ml-2 hidden w-14 shrink-0 text-right md:block">Response</div>
			<div class="w-5 shrink-0"></div>
		</div>
	{:else}
		<div class="mt-4 hidden items-center pb-2 text-[9px] font-medium uppercase tracking-wider text-muted-foreground sm:flex">
			<div class="w-5 shrink-0"></div>
			<div class="ml-2 min-w-0 flex-1">Service</div>
			<div class="hidden w-36 shrink-0 text-center md:block">Health (24h)</div>
			<div class="ml-2 hidden w-14 shrink-0 text-right md:block">Uptime</div>
			<div class="ml-3 hidden w-20 shrink-0 text-right md:block">Value</div>
			<div class="w-5 shrink-0"></div>
		</div>
	{/if}

	<!-- Rows -->
	<div class="divide-y divide-border/40">
		{#each monitors as m (m.id)}
			<a href="/monitors/{m.id}" class="group flex items-center py-3 transition-colors hover:bg-muted/30">
				<div class="w-5 shrink-0 flex justify-center">
					<StatusDot status={m.status === 'up' || m.status === 'down' || m.status === 'warn' ? m.status : 'unknown'} pulse={m.status === 'up'} />
				</div>
				<div class="flex-1 min-w-0 ml-2">
					<div class="flex items-center space-x-2">
						<span class="text-sm text-foreground truncate">{m.name}</span>
						<Pill tone="neutral">
							<span class="text-[9px] font-mono uppercase">{m.type}</span>
						</Pill>
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
							<Sparkline data={m.latencies} color={sparkColor(m.status)} />
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
</section>

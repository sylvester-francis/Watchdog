<script lang="ts">
	import { formatPercent, uptimeColor } from '$lib/utils';
	import { StatusPip } from '@sylvester-francis/watchdog-ui';
	import type { DashboardStats } from '$lib/types';

	interface Props {
		stats: DashboardStats;
		uptimePercent: number;
	}

	let { stats, uptimePercent }: Props = $props();

	let hasMonitors = $derived(stats.total_monitors > 0);
	let allUp = $derived(stats.monitors_down === 0 && hasMonitors);

	let statusTone = $derived<'success' | 'destructive' | 'muted'>(
		!hasMonitors ? 'muted' : allUp ? 'success' : 'destructive'
	);
	let statusLabel = $derived(
		!hasMonitors
			? 'No monitors configured'
			: allUp
				? 'All systems operational'
				: `${stats.monitors_down} system${stats.monitors_down > 1 ? 's' : ''} degraded`
	);
</script>

<section class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-4">
	<div class="min-w-0">
		<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
			<StatusPip tone={statusTone} label="Fleet status" />
			<span class="uppercase tracking-wider">Fleet</span>
		</div>
		<h1 class="mt-1.5 text-2xl font-medium text-foreground sm:text-3xl">{statusLabel}</h1>
		{#if hasMonitors}
			<p class="mt-1 font-mono tabular-nums text-sm text-muted-foreground">
				{stats.monitors_up} of {stats.total_monitors} monitors healthy · Last 24 hours
			</p>
		{:else}
			<p class="mt-1 text-sm text-muted-foreground">
				Deploy an agent and create monitors to get started.
			</p>
		{/if}
	</div>
	{#if hasMonitors}
		<div class="shrink-0">
			<span class="font-mono tabular-nums text-4xl tracking-tight {uptimeColor(uptimePercent)} sm:text-5xl">
				{formatPercent(uptimePercent)}<span class="text-base text-muted-foreground">%</span>
			</span>
		</div>
	{/if}
</section>

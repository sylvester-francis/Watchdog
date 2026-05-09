<script lang="ts">
	import { formatPercent, uptimeColor } from '$lib/utils';
	import StatBlock from '$lib/ui/StatBlock.svelte';
	import type { DashboardStats } from '$lib/types';

	interface Props {
		stats: DashboardStats;
		uptimePercent: number;
	}

	let { stats, uptimePercent }: Props = $props();

	const downAccent = $derived(stats.monitors_down > 0 ? 'warn' : 'neutral');
	const incidentsAccent = $derived(stats.active_incidents > 0 ? 'warn' : 'neutral');
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
	<StatBlock label="Monitors" value={stats.total_monitors} accent="neutral">
		<span data-stat-key="monitors" class="text-xs text-muted-foreground">{stats.monitors_up} up &middot; {stats.monitors_down} down</span>
	</StatBlock>

	<StatBlock label="Healthy" value={stats.monitors_up} accent="up">
		{#if stats.total_monitors > 0}
			<span data-stat-key="healthy" class="text-xs {uptimeColor(uptimePercent)}" style="opacity: 0.7">{formatPercent(uptimePercent)}% uptime (24h)</span>
		{:else}
			<span data-stat-key="healthy" class="text-xs text-muted-foreground">No checks yet</span>
		{/if}
	</StatBlock>

	<StatBlock label="Down" value={stats.monitors_down} accent={downAccent}>
		<span data-stat-key="down" class="text-xs text-muted-foreground">{stats.monitors_down > 0 ? 'Requires attention' : 'All clear'}</span>
	</StatBlock>

	<StatBlock label="Incidents" value={stats.active_incidents} accent={incidentsAccent}>
		<span data-stat-key="incidents" class="text-xs text-muted-foreground">{stats.active_incidents > 0 ? 'Active now' : 'No open incidents'}</span>
	</StatBlock>
</div>

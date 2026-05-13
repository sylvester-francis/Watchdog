<script lang="ts">
	import { formatPercent, uptimeColor } from '$lib/utils';
	import { StatCell, StatGrid } from '@sylvester-francis/watchdog-ui';
	import type { DashboardStats } from '$lib/types';

	interface Props {
		stats: DashboardStats;
		uptimePercent: number;
	}

	let { stats, uptimePercent }: Props = $props();
</script>

<StatGrid columns={4}>
	<StatCell label="Monitors" value={stats.total_monitors}>
		{#snippet sublabel()}
			<span data-stat-key="monitors">{stats.monitors_up} up · {stats.monitors_down} down</span>
		{/snippet}
	</StatCell>

	<StatCell label="Healthy" value={stats.monitors_up} valueTone="success">
		{#snippet sublabel()}
			{#if stats.total_monitors > 0}
				<span data-stat-key="healthy" class={uptimeColor(uptimePercent)}>{formatPercent(uptimePercent)}% uptime (24h)</span>
			{:else}
				<span data-stat-key="healthy">No checks yet</span>
			{/if}
		{/snippet}
	</StatCell>

	<StatCell
		label="Down"
		value={stats.monitors_down}
		valueTone={stats.monitors_down > 0 ? 'destructive' : 'default'}
	>
		{#snippet sublabel()}
			<span data-stat-key="down">{stats.monitors_down > 0 ? 'Requires attention' : 'All clear'}</span>
		{/snippet}
	</StatCell>

	<StatCell
		label="Incidents"
		value={stats.active_incidents}
		valueTone={stats.active_incidents > 0 ? 'warning' : 'default'}
	>
		{#snippet sublabel()}
			<span data-stat-key="incidents">{stats.active_incidents > 0 ? 'Active now' : 'No open incidents'}</span>
		{/snippet}
	</StatCell>
</StatGrid>

<script lang="ts">
	import { formatPercent, uptimeColor } from '$lib/utils';
	import type { DashboardStats } from '$lib/types';

	interface Props {
		stats: DashboardStats;
		uptimePercent: number;
	}

	let { stats, uptimePercent }: Props = $props();

	const cellClass = 'flex flex-col bg-background px-4 py-3.5';
	const labelClass = 'text-[11px] font-medium uppercase tracking-wider text-muted-foreground';
	const valueClass = 'mt-1 font-mono tabular-nums text-lg text-foreground';

	let downColorClass = $derived(stats.monitors_down > 0 ? 'text-destructive' : '');
	let incidentsColorClass = $derived(stats.active_incidents > 0 ? 'text-warning' : '');
</script>

<section class="grid grid-cols-2 gap-px overflow-hidden border-y border-border bg-border sm:grid-cols-4">
	<div class={cellClass}>
		<div class={labelClass}>Monitors</div>
		<div class={valueClass}>{stats.total_monitors}</div>
		<div data-stat-key="monitors" class="mt-0.5 font-mono tabular-nums text-[11px] text-muted-foreground">
			{stats.monitors_up} up · {stats.monitors_down} down
		</div>
	</div>

	<div class={cellClass}>
		<div class={labelClass}>Healthy</div>
		<div class="{valueClass} text-success">{stats.monitors_up}</div>
		{#if stats.total_monitors > 0}
			<div data-stat-key="healthy" class="mt-0.5 font-mono tabular-nums text-[11px] {uptimeColor(uptimePercent)}">
				{formatPercent(uptimePercent)}% uptime (24h)
			</div>
		{:else}
			<div data-stat-key="healthy" class="mt-0.5 text-[11px] text-muted-foreground">No checks yet</div>
		{/if}
	</div>

	<div class={cellClass}>
		<div class={labelClass}>Down</div>
		<div class="{valueClass} {downColorClass}">{stats.monitors_down}</div>
		<div data-stat-key="down" class="mt-0.5 text-[11px] text-muted-foreground">
			{stats.monitors_down > 0 ? 'Requires attention' : 'All clear'}
		</div>
	</div>

	<div class={cellClass}>
		<div class={labelClass}>Incidents</div>
		<div class="{valueClass} {incidentsColorClass}">{stats.active_incidents}</div>
		<div data-stat-key="incidents" class="mt-0.5 text-[11px] text-muted-foreground">
			{stats.active_incidents > 0 ? 'Active now' : 'No open incidents'}
		</div>
	</div>
</section>

<script lang="ts">
	import { Activity, CheckCircle2, XCircle, AlertTriangle } from 'lucide-svelte';
	import { formatPercent, uptimeColor } from '$lib/utils';
	import type { DashboardStats } from '$lib/types';

	interface Props {
		stats: DashboardStats;
		uptimePercent: number;
	}

	let { stats, uptimePercent }: Props = $props();
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
	<!-- Monitors -->
	<div class="bg-card border border-border rounded-lg p-3 border-l-[3px] border-l-muted">
		<div class="flex items-center justify-between mb-2">
			<p class="text-xs font-medium text-muted-foreground">Monitors</p>
			<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
				<Activity class="w-3 h-3 text-muted-foreground" />
			</div>
		</div>
		<p class="text-3xl text-foreground font-mono">{stats.total_monitors}</p>
		<p class="text-xs text-muted-foreground mt-1">{stats.monitors_up} up &middot; {stats.monitors_down} down</p>
	</div>

	<!-- Healthy -->
	<div class="bg-card border border-border rounded-lg p-3 border-l-[3px] border-l-emerald-500">
		<div class="flex items-center justify-between mb-2">
			<p class="text-xs font-medium text-muted-foreground">Healthy</p>
			<div class="w-6 h-6 bg-emerald-500/10 rounded flex items-center justify-center">
				<CheckCircle2 class="w-3 h-3 text-emerald-400" />
			</div>
		</div>
		<p class="text-3xl text-emerald-400 font-mono">{stats.monitors_up}</p>
		{#if stats.total_monitors > 0}
			<p class="text-xs mt-1 {uptimeColor(uptimePercent)}" style="opacity: 0.7">{formatPercent(uptimePercent)}% uptime (24h)</p>
		{:else}
			<p class="text-xs text-muted-foreground mt-1">No checks yet</p>
		{/if}
	</div>

	<!-- Down -->
	<div class="bg-card border {stats.monitors_down > 0 ? 'border-red-500/30' : 'border-border'} rounded-lg p-3 border-l-[3px] border-l-red-500">
		<div class="flex items-center justify-between mb-2">
			<p class="text-xs font-medium text-muted-foreground">Down</p>
			<div class="w-6 h-6 {stats.monitors_down > 0 ? 'bg-red-500/10' : 'bg-muted/50'} rounded flex items-center justify-center">
				<XCircle class="w-3 h-3 {stats.monitors_down > 0 ? 'text-red-400' : 'text-muted-foreground'}" />
			</div>
		</div>
		<p class="text-3xl {stats.monitors_down > 0 ? 'text-red-400' : 'text-foreground'} font-mono">{stats.monitors_down}</p>
		<p class="text-xs text-muted-foreground mt-1">{stats.monitors_down > 0 ? 'Requires attention' : 'All clear'}</p>
	</div>

	<!-- Incidents -->
	<div class="bg-card border {stats.active_incidents > 0 ? 'border-red-500/30' : 'border-border'} rounded-lg p-3 border-l-[3px] border-l-amber-500">
		<div class="flex items-center justify-between mb-2">
			<p class="text-xs font-medium text-muted-foreground">Incidents</p>
			<div class="w-6 h-6 {stats.active_incidents > 0 ? 'bg-amber-500/10' : 'bg-muted/50'} rounded flex items-center justify-center">
				<AlertTriangle class="w-3 h-3 {stats.active_incidents > 0 ? 'text-amber-400' : 'text-muted-foreground'}" />
			</div>
		</div>
		<div class="flex items-center space-x-2">
			<p class="text-3xl {stats.active_incidents > 0 ? 'text-red-400' : 'text-foreground'} font-mono">{stats.active_incidents}</p>
			{#if stats.active_incidents > 0}
				<span class="w-2 h-2 rounded-full bg-red-400 animate-pulse"></span>
			{/if}
		</div>
		<p class="text-xs text-muted-foreground mt-1">{stats.active_incidents > 0 ? 'Active now' : 'No open incidents'}</p>
	</div>
</div>

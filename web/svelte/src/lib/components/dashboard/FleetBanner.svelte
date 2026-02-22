<script lang="ts">
	import { MinusCircle } from 'lucide-svelte';
	import { formatPercent, uptimeColor } from '$lib/utils';
	import type { DashboardStats } from '$lib/types';

	interface Props {
		stats: DashboardStats;
		uptimePercent: number;
	}

	let { stats, uptimePercent }: Props = $props();

	let hasMonitors = $derived(stats.total_monitors > 0);
	let allUp = $derived(stats.monitors_down === 0 && hasMonitors);
	let accentClass = $derived(
		allUp ? 'bg-emerald-500' : stats.monitors_down > 0 ? 'bg-red-500' : 'bg-muted'
	);
</script>

<div class="bg-card border border-border rounded-lg mb-4 overflow-hidden">
	<div class="flex items-stretch">
		<div class="w-[3px] shrink-0 {accentClass}"></div>
		<div class="flex-1 px-4 py-3.5">
			<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
				{#if hasMonitors}
					<div class="flex items-center space-x-4">
						<span class="text-3xl font-mono tracking-tight {uptimeColor(uptimePercent)}">
							{formatPercent(uptimePercent)}<span class="text-base font-semibold text-muted-foreground">%</span>
						</span>
						<div>
							<h2 class="text-sm font-medium text-foreground">
								{#if allUp}
									All systems operational
								{:else}
									{stats.monitors_down} system{stats.monitors_down > 1 ? 's' : ''} degraded
								{/if}
							</h2>
							<p class="text-xs text-muted-foreground mt-0.5">
								{stats.monitors_up} of {stats.total_monitors} monitors healthy &middot; Last 24 hours
							</p>
						</div>
					</div>
					<div class="flex items-center space-x-1.5">
						<span class="text-xs font-mono px-1.5 py-0.5 rounded bg-emerald-500/15 text-emerald-400">{stats.monitors_up} up</span>
						{#if stats.monitors_down > 0}
							<span class="text-xs font-mono px-1.5 py-0.5 rounded bg-red-500/15 text-red-400">{stats.monitors_down} dn</span>
						{/if}
					</div>
				{:else}
					<div class="flex items-center space-x-3">
						<div class="w-8 h-8 rounded-md bg-muted/50 flex items-center justify-center shrink-0">
							<MinusCircle class="w-4 h-4 text-muted-foreground" />
						</div>
						<div>
							<h2 class="text-sm font-medium text-foreground">No monitors configured</h2>
							<p class="text-xs text-muted-foreground mt-0.5">Deploy an agent and create monitors to get started</p>
						</div>
					</div>
				{/if}
			</div>
		</div>
	</div>
</div>

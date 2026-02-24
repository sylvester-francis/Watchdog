<script lang="ts">
	import { AlertCircle, Clock, CheckCircle2, ArrowRight } from 'lucide-svelte';
	import { formatTimeAgo, formatDuration } from '$lib/utils';
	import type { Incident, MonitorSummary } from '$lib/types';

	interface Props {
		incidents: Incident[];
		monitors: Map<string, MonitorSummary>;
	}

	let { incidents, monitors }: Props = $props();

	let displayed = $derived(incidents.slice(0, 5));
</script>

<div class="bg-card border border-border rounded-lg self-start">
	<div class="px-4 py-3 border-b border-border flex items-center justify-between">
		<div class="flex items-center space-x-2">
			<h2 class="text-sm font-medium text-foreground">Active Incidents</h2>
		</div>
		{#if incidents.length > 0}
			<a href="/incidents" class="text-xs text-muted-foreground hover:text-foreground transition-colors flex items-center space-x-1">
				<span>View all</span>
				<ArrowRight class="w-3 h-3" />
			</a>
		{/if}
	</div>

	{#if displayed.length > 0}
		<div class="divide-y divide-border/30">
			{#each displayed as incident (incident.id)}
				{@const monitor = monitors.get(incident.monitor_id)}
				<div class="px-4 py-2.5 flex items-center justify-between transition-colors hover:bg-card-elevated">
					<div class="flex items-center space-x-3">
						<div class="w-7 h-7 rounded-md {incident.status === 'open' ? 'bg-red-500/10' : 'bg-yellow-500/10'} flex items-center justify-center">
							{#if incident.status === 'open'}
								<AlertCircle class="w-3.5 h-3.5 text-red-400" />
							{:else}
								<Clock class="w-3.5 h-3.5 text-yellow-400" />
							{/if}
						</div>
						<div>
							<p class="text-sm font-medium text-foreground">{monitor?.name ?? 'Unknown Monitor'}</p>
							<p class="text-[10px] text-muted-foreground">{formatTimeAgo(incident.started_at)} &middot; <span class="font-mono">{formatDuration(incident.started_at)}</span></p>
						</div>
					</div>
					<span class="text-[10px] px-1.5 py-0.5 rounded font-mono
						{incident.status === 'open' ? 'bg-red-500/15 text-red-400' : 'bg-yellow-500/15 text-yellow-400'}">
						{incident.status}
					</span>
				</div>
			{/each}
		</div>
	{:else}
		<div class="text-center py-8">
			<div class="w-12 h-12 bg-emerald-500/5 rounded-lg flex items-center justify-center mx-auto mb-3">
				<CheckCircle2 class="w-6 h-6 text-emerald-400/40" />
			</div>
			<p class="text-sm text-muted-foreground">No active incidents</p>
			<a href="/incidents?status=all" class="text-xs text-muted-foreground/60 hover:text-muted-foreground mt-1 inline-block transition-colors">View incident history</a>
		</div>
	{/if}
</div>

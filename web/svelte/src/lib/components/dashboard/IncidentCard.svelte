<script lang="ts">
	import { ArrowRight } from 'lucide-svelte';
	import { formatTimeAgo, formatDuration } from '$lib/utils';
	import { SectionHeader, StatusPip } from '@sylvester-francis/watchdog-ui';
	import type { Incident, MonitorSummary } from '$lib/types';

	interface Props {
		incidents: Incident[];
		monitors: Map<string, MonitorSummary>;
	}

	let { incidents, monitors }: Props = $props();

	let displayed = $derived(incidents.slice(0, 5));
</script>

<section>
	<SectionHeader title="Active Incidents">
		{#snippet action()}
			{#if incidents.length > 0}
				<a href="/incidents" class="flex items-center gap-1 text-xs text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline">
					<span>View all</span>
					<ArrowRight class="h-3 w-3" />
				</a>
			{/if}
		{/snippet}
	</SectionHeader>

	{#if displayed.length > 0}
		<div class="divide-y divide-border/40">
			{#each displayed as incident (incident.id)}
				{@const monitor = monitors.get(incident.monitor_id)}
				<div class="flex items-center justify-between gap-3 py-3 transition-colors hover:bg-muted/30">
					<div class="min-w-0">
						<div class="flex items-center gap-2">
							<StatusPip tone={incident.status === 'open' ? 'destructive' : 'warning'} />
							<p class="truncate text-sm text-foreground">{monitor?.name ?? incident.monitor_name ?? 'Unknown Monitor'}</p>
						</div>
						<p class="mt-0.5 ml-3.5 font-mono tabular-nums text-[11px] text-muted-foreground">
							{formatTimeAgo(incident.started_at)} · {formatDuration(incident.started_at)}
						</p>
					</div>
					<span class="shrink-0 font-mono tabular-nums text-[11px] uppercase tracking-wider {incident.status === 'open' ? 'text-destructive' : 'text-warning'}">
						{incident.status}
					</span>
				</div>
			{/each}
		</div>
	{:else}
		<div class="pt-6 text-center">
			<p class="font-mono tabular-nums text-xs uppercase tracking-wider text-muted-foreground">
				<StatusPip tone="success" />
				No active incidents
			</p>
			<a href="/incidents?status=all" class="mt-2 inline-block text-xs text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline">
				View incident history
			</a>
		</div>
	{/if}
</section>

<script lang="ts">
	import { onMount } from 'svelte';
	import { getMonitorAnomalies, type LatencyAnomaly } from '$lib/api/monitors';

	interface Props {
		monitorId: string;
	}
	let { monitorId }: Props = $props();

	let anomalies = $state<LatencyAnomaly[]>([]);
	let loading = $state(true);
	let loadError = $state('');

	onMount(async () => {
		try {
			const res = await getMonitorAnomalies(monitorId, 86400); // 24h
			anomalies = res.data ?? [];
		} catch (e) {
			loadError = e instanceof Error ? e.message : 'Failed to load anomalies';
		} finally {
			loading = false;
		}
	});

	function formatTime(iso: string): string {
		return new Date(iso).toLocaleString([], {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function methodLabel(m: string): string {
		switch (m) {
			case 'zscore':
				return 'Z-SCORE';
			case 'iqr':
				return 'IQR';
			case 'both':
				return 'BOTH';
			default:
				return m.toUpperCase();
		}
	}
</script>

<section class="mt-8">
	<div class="flex items-center justify-between border-b border-border pb-3 mb-3">
		<h3 class="text-sm font-medium text-foreground">Anomalies (last 24h)</h3>
		{#if !loading && anomalies.length > 0}
			<span class="text-xs font-mono uppercase text-muted-foreground tabular-nums">
				{anomalies.length} detected
			</span>
		{/if}
	</div>

	{#if loading}
		<p class="text-xs text-muted-foreground">Loading…</p>
	{:else if loadError}
		<p class="text-xs text-destructive">{loadError}</p>
	{:else if anomalies.length === 0}
		<p class="text-xs text-muted-foreground">No anomalies detected. Latency has stayed within the normal range over the last 24 hours.</p>
	{:else}
		<div class="overflow-hidden border-y border-border">
			{#each anomalies.slice(0, 20) as a}
				<div class="flex items-center gap-3 px-1 py-2 border-b border-border last:border-b-0 text-xs">
					<span class="inline-block h-1.5 w-1.5 rounded-full bg-warning flex-shrink-0"></span>
					<span class="font-mono tabular-nums text-muted-foreground w-32">{formatTime(a.time)}</span>
					<span class="font-mono tabular-nums text-foreground flex-1">{a.latency_ms.toLocaleString()} ms</span>
					<span class="font-mono uppercase text-muted-foreground" title="Z-score: {a.z_score.toFixed(2)}">
						{methodLabel(a.method)}
					</span>
				</div>
			{/each}
			{#if anomalies.length > 20}
				<p class="text-[10px] text-muted-foreground text-center pt-2">…and {anomalies.length - 20} more</p>
			{/if}
		</div>
	{/if}
</section>

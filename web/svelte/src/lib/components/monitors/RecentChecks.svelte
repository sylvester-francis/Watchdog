<script lang="ts">
	import { onMount } from 'svelte';
	import { Loader2 } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { formatTimeAgo } from '$lib/utils';
	import { getToasts } from '$lib/stores/toast';
	import type { HeartbeatPoint } from '$lib/types';

	interface Props {
		monitorId: string;
	}

	let { monitorId }: Props = $props();

	const toast = getToasts();

	let heartbeats = $state<HeartbeatPoint[]>([]);
	let loading = $state(true);

	function statusBadgeClass(status: string): string {
		switch (status) {
			case 'up':
				return 'bg-emerald-500/15 text-emerald-400 border-emerald-500/20';
			case 'down':
				return 'bg-red-500/15 text-red-400 border-red-500/20';
			case 'timeout':
				return 'bg-amber-500/15 text-amber-400 border-amber-500/20';
			case 'error':
				return 'bg-red-500/15 text-red-400 border-red-500/20';
			default:
				return 'bg-muted/50 text-muted-foreground border-border';
		}
	}

	function statusLabel(status: string): string {
		switch (status) {
			case 'up': return 'Up';
			case 'down': return 'Down';
			case 'timeout': return 'Timeout';
			case 'error': return 'Error';
			default: return status;
		}
	}

	function formatLatency(ms: number | null): string {
		if (ms === null || ms === undefined) return '--';
		if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
		return `${ms}ms`;
	}

	async function fetchHeartbeats() {
		loading = true;
		try {
			const res = await monitorsApi.getHeartbeats(monitorId);
			// Show most recent first, limit to 20
			heartbeats = (res.data ?? []).slice(-20).reverse();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load heartbeats');
			heartbeats = [];
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchHeartbeats();
	});
</script>

<div class="bg-card border border-border rounded-lg">
	<div class="px-4 py-3 border-b border-border">
		<h2 class="text-sm font-medium text-foreground">Recent Checks</h2>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<Loader2 class="w-5 h-5 text-muted-foreground animate-spin" />
		</div>
	{:else if heartbeats.length === 0}
		<div class="p-8 text-center">
			<p class="text-xs text-muted-foreground">No heartbeat data yet</p>
		</div>
	{:else}
		<!-- Column headers -->
		<div class="hidden sm:grid grid-cols-[1fr_80px_90px] items-center px-4 py-2 border-b border-border/30 text-[9px] font-medium text-muted-foreground uppercase tracking-wider">
			<div>Time</div>
			<div class="text-center">Status</div>
			<div class="text-right">Latency</div>
		</div>

		<div class="divide-y divide-border/20 max-h-[480px] overflow-y-auto">
			{#each heartbeats as hb, i}
				<div class="grid grid-cols-[1fr_80px_90px] items-center px-4 py-2.5 hover:bg-card-elevated transition-colors">
					<!-- Time -->
					<div class="text-xs text-muted-foreground font-mono">
						{formatTimeAgo(hb.time)}
					</div>

					<!-- Status badge -->
					<div class="flex justify-center">
						<span class="inline-flex items-center px-2 py-0.5 text-[10px] font-medium rounded border {statusBadgeClass(hb.status)}">
							{statusLabel(hb.status)}
						</span>
					</div>

					<!-- Latency -->
					<div class="text-right">
						<span class="text-xs font-mono {hb.status === 'up' ? 'text-foreground' : 'text-muted-foreground'}">
							{formatLatency(hb.latency_ms)}
						</span>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

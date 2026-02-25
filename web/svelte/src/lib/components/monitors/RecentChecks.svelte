<script lang="ts">
	import { onMount } from 'svelte';
	import { Loader2, List } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { formatTimeAgo } from '$lib/utils';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { HeartbeatPoint } from '$lib/types';

	interface Props {
		monitorId: string;
		monitorType?: string;
	}

	let { monitorId, monitorType = '' }: Props = $props();

	const toast = getToasts();
	const isNonLatency = $derived(monitorType === 'system' || monitorType === 'docker' || monitorType === 'service');
	const isTLS = $derived(monitorType === 'tls');

	let heartbeats = $state<HeartbeatPoint[]>([]);
	let loading = $state(true);

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

	function parseMetricValue(msg: string | undefined, status?: string): string {
		if (!msg) {
			if (status === 'up') return 'Running';
			if (status === 'down') return 'Stopped';
			return '--';
		}
		const match = msg.match(/([\d.]+)%/);
		if (match) return `${parseFloat(match[1]).toFixed(1)}%`;
		// Service/docker: show simple status instead of raw message
		if (status === 'up') return 'Running';
		if (status === 'down') return 'Stopped';
		return msg;
	}

	async function fetchHeartbeats() {
		loading = true;
		try {
			const res = await monitorsApi.getHeartbeats(monitorId);
			const arr = Array.isArray(res) ? res : [];
			heartbeats = arr.slice(-20).reverse();
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
	<div class="px-5 py-3.5 border-b border-border flex items-center space-x-2">
		<List class="w-4 h-4 text-muted-foreground" />
		<h3 class="text-sm font-medium text-foreground">Recent Checks</h3>
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
		<div class="overflow-x-auto">
			<table class="w-full">
				<thead>
					<tr class="border-b border-border">
						<th class="px-5 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Time</th>
						<th class="px-5 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Status</th>
						<th class="px-5 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
							{isNonLatency ? 'Value' : 'Latency'}
						</th>
						<th class="px-5 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
							{isNonLatency ? 'Detail' : isTLS ? 'Certificate' : 'Status'}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border/30">
					{#each heartbeats as hb}
						<tr class="hover:bg-card-elevated transition-colors">
							<td class="px-5 py-2.5">
								<span class="text-xs text-muted-foreground font-mono">{formatTimeAgo(hb.time)}</span>
							</td>
							<td class="px-5 py-2.5">
								<div class="flex items-center space-x-2">
									<div class="w-1.5 h-1.5 rounded-full {hb.status === 'up' ? 'bg-emerald-400' : 'bg-red-400'}"></div>
									<span class="text-xs font-mono {hb.status === 'up' ? 'text-emerald-400' : 'text-red-400'}">{statusLabel(hb.status)}</span>
								</div>
							</td>
							<td class="px-5 py-2.5">
								{#if isNonLatency}
									<span class="text-xs text-foreground font-mono">{parseMetricValue(hb.error_message, hb.status)}</span>
								{:else if hb.latency_ms != null}
									<span class="text-xs text-foreground font-mono">{formatLatency(hb.latency_ms)}</span>
								{:else}
									<span class="text-xs text-muted-foreground">--</span>
								{/if}
							</td>
							<td class="px-5 py-2.5">
								{#if isTLS && hb.cert_expiry_days != null}
									{@const days = hb.cert_expiry_days}
									<span class="text-xs font-mono {days < 14 ? 'text-red-400' : days < 30 ? 'text-amber-400' : 'text-emerald-400'}">
										Expires in {days}d
									</span>
								{:else if isNonLatency && hb.error_message}
									<span class="text-xs text-muted-foreground font-mono truncate max-w-[200px] inline-block">{hb.error_message}</span>
								{:else if isNonLatency && hb.status === 'up'}
									<span class="text-xs text-emerald-400 font-mono">OK</span>
								{:else if hb.status === 'down' || hb.status === 'error'}
									<span class="text-xs text-red-400 font-mono">Check failed</span>
								{:else if hb.status === 'timeout'}
									<span class="text-xs text-amber-400 font-mono">Timeout</span>
								{:else if hb.status === 'up'}
									<span class="text-xs text-emerald-400 font-mono">OK</span>
								{:else}
									<span class="text-xs text-muted-foreground">{@html '&mdash;'}</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

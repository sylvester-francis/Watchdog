<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2 } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { formatTimeAgo } from '$lib/utils';

	function formatCheckTime(date: string): string {
		const d = new Date(date);
		const now = Date.now();
		const diffMs = now - d.getTime();
		const diffHours = diffMs / (1000 * 60 * 60);

		if (diffHours < 1) {
			return formatTimeAgo(date);
		}

		const today = new Date();
		const isToday = d.toDateString() === today.toDateString();
		const time = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
		if (isToday) return time;
		const dateStr = d.toLocaleDateString([], { month: 'short', day: 'numeric' });
		return `${dateStr} ${time}`;
	}
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
	const isPortScan = $derived(monitorType === 'port_scan');
	const isSNMP = $derived(monitorType === 'snmp');

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

	function parsePortCounts(hb: HeartbeatPoint): string {
		const msg = hb.error_message || '';
		const openMatch = msg.match(/(\d+)\s*open/i);
		const scannedMatch = msg.match(/(\d+)\s*scanned/i);
		if (openMatch && scannedMatch) return `${openMatch[1]}/${scannedMatch[1]}`;
		if (openMatch) return `${openMatch[1]} open`;
		return hb.status === 'up' ? 'OK' : '--';
	}

	function parsePortDetail(hb: HeartbeatPoint): { text: string; color: string } {
		const msg = hb.error_message || '';
		const missingMatch = msg.match(/missing:\s*([^;]+)/i);
		const unexpectedMatch = msg.match(/unexpected:\s*([^;]+)/i);
		if (missingMatch) return { text: `Missing: ${missingMatch[1].trim()}`, color: 'text-red-400' };
		if (unexpectedMatch) return { text: `Unexpected: ${unexpectedMatch[1].trim()}`, color: 'text-yellow-400' };
		if (hb.status === 'up') return { text: 'Clean', color: 'text-emerald-400' };
		return { text: 'Check failed', color: 'text-red-400' };
	}

	let initialLoad = true;

	async function fetchHeartbeats() {
		if (initialLoad) loading = true;
		try {
			const res = await monitorsApi.getHeartbeats(monitorId);
			const arr = Array.isArray(res) ? res : [];
			heartbeats = arr.slice(0, 20);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load heartbeats');
			heartbeats = [];
		} finally {
			loading = false;
			initialLoad = false;
		}
	}

	let pollInterval: ReturnType<typeof setInterval>;

	onMount(() => {
		fetchHeartbeats();
		pollInterval = setInterval(fetchHeartbeats, 30_000);
	});

	onDestroy(() => {
		clearInterval(pollInterval);
	});
</script>

<section>
	<div class="border-b border-border pb-3">
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
							{isPortScan ? 'Ports' : isSNMP ? 'Response' : isNonLatency ? 'Value' : 'Latency'}
						</th>
						<th class="px-5 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
							{isPortScan ? 'Detail' : isSNMP ? 'Value' : isNonLatency ? 'Detail' : isTLS ? 'Certificate' : 'Status'}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border/30">
					{#each heartbeats as hb}
						<tr class="hover:bg-card-elevated transition-colors">
							<td class="px-5 py-2.5">
								<span class="text-xs text-muted-foreground font-mono">{formatCheckTime(hb.time)}</span>
							</td>
							<td class="px-5 py-2.5">
								<div class="flex items-center space-x-2">
									<div class="w-1.5 h-1.5 rounded-full {hb.status === 'up' ? 'bg-emerald-400' : 'bg-red-400'}"></div>
									<span class="text-xs font-mono {hb.status === 'up' ? 'text-emerald-400' : 'text-red-400'}">{statusLabel(hb.status)}</span>
								</div>
							</td>
							<td class="px-5 py-2.5">
								{#if isSNMP}
									<span class="text-xs text-foreground font-mono">{formatLatency(hb.latency_ms)}</span>
								{:else if isPortScan}
									<span class="text-xs text-foreground font-mono">{parsePortCounts(hb)}</span>
								{:else if isNonLatency}
									<span class="text-xs text-foreground font-mono">{parseMetricValue(hb.error_message, hb.status)}</span>
								{:else if hb.latency_ms != null}
									<span class="text-xs text-foreground font-mono">{formatLatency(hb.latency_ms)}</span>
								{:else}
									<span class="text-xs text-muted-foreground">--</span>
								{/if}
							</td>
							<td class="px-5 py-2.5">
								{#if isSNMP && hb.status === 'up' && hb.error_message}
									<span class="text-xs text-emerald-400 font-mono truncate max-w-[250px] inline-block" title={hb.error_message}>{hb.error_message}</span>
								{:else if isSNMP && hb.status === 'up'}
									<span class="text-xs text-emerald-400 font-mono">OK</span>
								{:else if isSNMP && hb.error_message}
									<span class="text-xs text-red-400 font-mono truncate max-w-[200px] inline-block">{hb.error_message}</span>
								{:else if isPortScan}
									{@const detail = parsePortDetail(hb)}
									<span class="text-xs font-mono truncate max-w-[200px] inline-block {detail.color}">{detail.text}</span>
								{:else if isTLS && hb.cert_expiry_days != null}
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
</section>

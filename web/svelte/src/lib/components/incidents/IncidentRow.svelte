<script lang="ts">
	import { base } from '$app/paths';
	import { Loader2 } from 'lucide-svelte';
	import { formatTimeAgo, formatDuration } from '$lib/utils';
	import type { Incident, MonitorSummary } from '$lib/types';

	interface Props {
		incident: Incident;
		monitor: MonitorSummary | undefined;
		onAcknowledge: (id: string) => Promise<void>;
		onResolve: (id: string) => Promise<void>;
	}

	let { incident, monitor, onAcknowledge, onResolve }: Props = $props();

	let ackLoading = $state(false);
	let resolveLoading = $state(false);

	function statusBadgeClass(status: string): string {
		switch (status) {
			case 'open':
				return 'bg-red-500/10 text-red-400 border border-red-500/20';
			case 'acknowledged':
				return 'bg-yellow-500/10 text-yellow-400 border border-yellow-500/20';
			case 'resolved':
				return 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20';
			default:
				return 'bg-muted/50 text-muted-foreground border border-border';
		}
	}

	function formatTTR(seconds: number | null): string {
		if (seconds === null) return '--';
		if (seconds < 60) return `${seconds}s`;
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);
		if (hours === 0) return `${minutes}m`;
		return `${hours}h ${minutes % 60}m`;
	}

	async function handleAcknowledge() {
		ackLoading = true;
		try {
			await onAcknowledge(incident.id);
		} finally {
			ackLoading = false;
		}
	}

	async function handleResolve() {
		resolveLoading = true;
		try {
			await onResolve(incident.id);
		} finally {
			resolveLoading = false;
		}
	}
</script>

<tr class="border-b border-border/20 hover:bg-card-elevated transition-colors">
	<!-- Status badge -->
	<td class="px-4 py-3">
		<span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-medium capitalize {statusBadgeClass(incident.status)}">
			{incident.status}
		</span>
	</td>

	<!-- Monitor name + type -->
	<td class="px-4 py-3">
		{#if monitor}
			<a href="{base}/monitors/{incident.monitor_id}" class="group">
				<div class="flex items-center space-x-2">
					<span class="text-sm text-foreground group-hover:text-accent transition-colors">{monitor.name}</span>
					<span class="text-[9px] text-muted-foreground font-mono uppercase shrink-0 px-1.5 py-0.5 rounded bg-muted/50">{monitor.type}</span>
				</div>
			</a>
		{:else}
			<div class="flex items-center space-x-2">
				<span class="text-sm text-muted-foreground">{incident.monitor_id.slice(0, 8)}...</span>
				<span class="text-[9px] text-muted-foreground font-mono uppercase shrink-0 px-1.5 py-0.5 rounded bg-muted/50">unknown</span>
			</div>
		{/if}
	</td>

	<!-- Target -->
	<td class="px-4 py-3 hidden lg:table-cell">
		<span class="text-xs font-mono text-muted-foreground truncate max-w-[200px] inline-block">
			{monitor?.target ?? '--'}
		</span>
	</td>

	<!-- Started -->
	<td class="px-4 py-3 hidden md:table-cell">
		<span class="text-xs text-muted-foreground">{formatTimeAgo(incident.started_at)}</span>
	</td>

	<!-- Duration / TTR -->
	<td class="px-4 py-3 hidden md:table-cell">
		{#if incident.status === 'resolved'}
			<span class="text-xs font-mono text-emerald-400">{formatTTR(incident.ttr_seconds)}</span>
		{:else}
			<span class="text-xs font-mono text-muted-foreground">{formatDuration(incident.started_at)}</span>
		{/if}
	</td>

	<!-- Actions -->
	<td class="px-4 py-3">
		{#if incident.status !== 'resolved'}
			<div class="flex items-center space-x-2">
				{#if incident.status === 'open'}
					<button
						onclick={handleAcknowledge}
						disabled={ackLoading || resolveLoading}
						class="inline-flex items-center space-x-1 px-2 py-1 text-[10px] font-medium rounded border border-yellow-500/30 text-yellow-400 hover:bg-yellow-500/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{#if ackLoading}
							<Loader2 class="w-3 h-3 animate-spin" />
						{/if}
						<span>Acknowledge</span>
					</button>
				{/if}
				<button
					onclick={handleResolve}
					disabled={ackLoading || resolveLoading}
					class="inline-flex items-center space-x-1 px-2 py-1 text-[10px] font-medium rounded border border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{#if resolveLoading}
						<Loader2 class="w-3 h-3 animate-spin" />
					{/if}
					<span>Resolve</span>
				</button>
			</div>
		{/if}
	</td>
</tr>

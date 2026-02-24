<script lang="ts">
	import { Loader2, AlertTriangle, Eye, CheckCircle2 } from 'lucide-svelte';
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

<tr class="hover:bg-card-elevated transition-colors group">
	<!-- Status badge -->
	<td class="px-4 py-3.5">
		<div class="flex items-center space-x-2">
			<div class="w-2 h-2 rounded-full
				{incident.status === 'open' ? 'bg-red-400 animate-pulse' : incident.status === 'acknowledged' ? 'bg-yellow-400' : 'bg-emerald-400'}"
				aria-label="Status: {incident.status}"></div>
			<span class="text-xs font-medium px-1.5 py-0.5 rounded
				{statusBadgeClass(incident.status)}">
				{incident.status}
			</span>
		</div>
	</td>

	<!-- Monitor name + type with icon -->
	<td class="px-4 py-3.5">
		<div class="flex items-center space-x-2.5">
			<div class="w-7 h-7 rounded-md flex items-center justify-center shrink-0
				{incident.status === 'open' ? 'bg-red-500/10' : incident.status === 'acknowledged' ? 'bg-yellow-500/10' : 'bg-emerald-500/10'}">
				{#if incident.status === 'open'}
					<AlertTriangle class="w-3.5 h-3.5 text-red-400" />
				{:else if incident.status === 'acknowledged'}
					<Eye class="w-3.5 h-3.5 text-yellow-400" />
				{:else}
					<CheckCircle2 class="w-3.5 h-3.5 text-emerald-400" />
				{/if}
			</div>
			<div>
				{#if monitor}
					<a href="/monitors/{incident.monitor_id}" class="group/link">
						<span class="text-sm font-medium text-foreground group-hover/link:text-accent transition-colors">{monitor.name}</span>
						<span class="ml-1.5 text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground uppercase font-mono hidden lg:inline">{monitor.type}</span>
					</a>
				{:else}
					<span class="text-sm text-muted-foreground">{incident.monitor_id.slice(0, 8)}...</span>
				{/if}
			</div>
		</div>
	</td>

	<!-- Target -->
	<td class="px-4 py-3.5 hidden lg:table-cell">
		<span class="text-xs font-mono text-muted-foreground truncate max-w-[200px] inline-block">
			{monitor?.target ?? '--'}
		</span>
	</td>

	<!-- Started -->
	<td class="px-4 py-3.5 hidden md:table-cell">
		<span class="text-xs text-muted-foreground">{formatTimeAgo(incident.started_at)}</span>
	</td>

	<!-- Duration / TTR -->
	<td class="px-4 py-3.5 hidden md:table-cell">
		{#if incident.status === 'resolved'}
			<span class="text-xs font-mono text-muted-foreground">{formatTTR(incident.ttr_seconds)} TTR</span>
		{:else}
			<span class="text-xs font-mono text-muted-foreground">{formatDuration(incident.started_at)}</span>
		{/if}
	</td>

	<!-- Actions -->
	<td class="px-4 py-3.5">
		{#if incident.status !== 'resolved'}
			<div class="flex items-center space-x-1.5">
				{#if incident.status === 'open'}
					<button
						onclick={handleAcknowledge}
						disabled={ackLoading || resolveLoading}
						class="px-2 py-1 bg-yellow-500/10 text-yellow-400 hover:bg-yellow-500/20 rounded text-xs font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{#if ackLoading}
							<Loader2 class="w-3 h-3 animate-spin inline mr-1" />
						{/if}
						Ack
					</button>
				{/if}
				<button
					onclick={handleResolve}
					disabled={ackLoading || resolveLoading}
					class="px-2 py-1 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 rounded text-xs font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{#if resolveLoading}
						<Loader2 class="w-3 h-3 animate-spin inline mr-1" />
					{/if}
					Resolve
				</button>
			</div>
		{/if}
	</td>
</tr>

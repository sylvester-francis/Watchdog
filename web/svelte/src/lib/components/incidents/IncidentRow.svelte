<script lang="ts">
	import { Loader2, ChevronRight, Search } from 'lucide-svelte';
	import { formatTimeAgo, formatDuration } from '$lib/utils';
	import { StatusPip } from '@sylvester-francis/watchdog-ui';
	import type { Incident, MonitorSummary } from '$lib/types';

	type Tone = 'success' | 'destructive' | 'warning' | 'accent' | 'muted';

	function statusTone(status: string): Tone {
		if (status === 'open') return 'destructive';
		if (status === 'acknowledged') return 'warning';
		if (status === 'resolved') return 'success';
		return 'muted';
	}

	function statusTextClass(status: string): string {
		if (status === 'open') return 'text-destructive';
		if (status === 'acknowledged') return 'text-warning';
		if (status === 'resolved') return 'text-success';
		return 'text-muted-foreground';
	}

	interface Props {
		incident: Incident;
		monitor: MonitorSummary | undefined;
		onAcknowledge: (id: string) => Promise<void>;
		onResolve: (id: string) => Promise<void>;
		onInvestigate?: (id: string) => void;
		canWrite?: boolean;
	}

	let { incident, monitor, onAcknowledge, onResolve, onInvestigate, canWrite = true }: Props = $props();

	let ackLoading = $state(false);
	let resolveLoading = $state(false);

	function formatTTR(seconds: number | null | undefined): string {
		if (seconds == null) return '--';
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

<tr class="group cursor-pointer transition-colors hover:bg-muted/30" onclick={() => window.location.href = `/incidents/${incident.id}`}>
	<!-- Status -->
	<td class="py-3.5 pl-1 pr-4">
		<div class="flex items-center gap-2">
			<StatusPip tone={statusTone(incident.status)} pulse={incident.status === 'open'} label="Status: {incident.status}" />
			<span class="font-mono tabular-nums text-xs uppercase tracking-wider {statusTextClass(incident.status)}">
				{incident.status}
			</span>
		</div>
	</td>

	<!-- Monitor name + type -->
	<td class="px-4 py-3.5">
		{#if monitor}
			<a href="/incidents/{incident.id}" class="group/link" onclick={(e) => e.stopPropagation()}>
				<span class="text-sm font-medium text-foreground transition-colors group-hover/link:text-accent">{monitor.name}</span>
				<span class="ml-2 hidden font-mono tabular-nums text-[10px] uppercase tracking-wider text-muted-foreground lg:inline">{monitor.type}</span>
			</a>
		{:else}
			<a href="/incidents/{incident.id}" class="text-sm text-muted-foreground transition-colors hover:text-accent" onclick={(e) => e.stopPropagation()}>
				{incident.monitor_name || 'Unknown Monitor'}
			</a>
		{/if}
	</td>

	<!-- Target -->
	<td class="hidden px-4 py-3.5 lg:table-cell">
		<span class="inline-block max-w-[200px] truncate font-mono tabular-nums text-xs text-muted-foreground">
			{monitor?.target ?? '--'}
		</span>
	</td>

	<!-- Started -->
	<td class="hidden px-4 py-3.5 md:table-cell">
		<span class="font-mono tabular-nums text-xs text-muted-foreground">{formatTimeAgo(incident.started_at)}</span>
	</td>

	<!-- Duration / TTR -->
	<td class="hidden px-4 py-3.5 md:table-cell">
		{#if incident.status === 'resolved'}
			<span class="font-mono tabular-nums text-xs text-muted-foreground">{formatTTR(incident.ttr_seconds)} TTR</span>
		{:else}
			<span class="font-mono tabular-nums text-xs text-muted-foreground">{formatDuration(incident.started_at)}</span>
		{/if}
	</td>

	<!-- Actions -->
	<td class="px-1 py-3.5 pr-1 sm:px-4" onclick={(e) => e.stopPropagation()}>
		<div class="flex items-center justify-end gap-2 sm:gap-3">
			{#if onInvestigate}
				<button
					onclick={() => onInvestigate(incident.id)}
					class="inline-flex min-h-[36px] items-center gap-1 -my-1.5 px-1 text-xs text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
					title="Investigate"
				>
					<Search class="h-3 w-3" />
					<span>Investigate</span>
				</button>
			{/if}
			{#if incident.status !== 'resolved' && canWrite}
				{#if incident.status === 'open'}
					<button
						onclick={handleAcknowledge}
						disabled={ackLoading || resolveLoading}
						class="inline-flex min-h-[36px] items-center gap-1 -my-1.5 px-1 text-xs text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline disabled:opacity-50"
					>
						{#if ackLoading}<Loader2 class="h-3 w-3 animate-spin" />{/if}
						Ack
					</button>
				{/if}
				<button
					onclick={handleResolve}
					disabled={ackLoading || resolveLoading}
					class="inline-flex min-h-[36px] items-center gap-1 -my-1.5 px-1 text-xs text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline disabled:opacity-50"
				>
					{#if resolveLoading}<Loader2 class="h-3 w-3 animate-spin" />{/if}
					Resolve
				</button>
			{/if}
			<ChevronRight class="hidden h-3 w-3 text-muted-foreground/30 opacity-0 transition-opacity group-hover:opacity-100 sm:block" />
		</div>
	</td>
</tr>

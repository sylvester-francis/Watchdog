<script lang="ts">
	import { Server, History, Cpu, Shield } from 'lucide-svelte';
	import type { IncidentInvestigation } from '$lib/types';
	import IncidentTimeline from './IncidentTimeline.svelte';

	interface Props {
		investigation: IncidentInvestigation;
	}

	let { investigation }: Props = $props();

	function patternLabel(pattern: string): string {
		switch (pattern) {
			case 'first_time': return 'First Occurrence';
			case 'recurring': return 'Recurring';
			case 'frequent': return 'Frequent';
			default: return pattern;
		}
	}

	function patternColor(pattern: string): string {
		switch (pattern) {
			case 'first_time': return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
			case 'recurring': return 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20';
			case 'frequent': return 'bg-red-500/10 text-red-400 border-red-500/20';
			default: return 'bg-muted/50 text-muted-foreground border-border';
		}
	}

	function formatMTTR(seconds: number | null): string {
		if (seconds == null) return '--';
		if (seconds < 60) return `${seconds}s`;
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);
		if (hours === 0) return `${minutes}m`;
		return `${hours}h ${minutes % 60}m`;
	}

	function formatTimeAgo(iso: string): string {
		const diff = Date.now() - new Date(iso).getTime();
		const minutes = Math.floor(diff / 60000);
		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		return `${days}d ago`;
	}
</script>

<div class="space-y-5">
	<!-- Overview Cards -->
	<div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
		<!-- Pattern -->
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-2">Pattern</div>
			<span class="text-xs font-medium px-2 py-0.5 rounded border {patternColor(investigation.recurrence_pattern ?? 'unknown')}">
				{patternLabel(investigation.recurrence_pattern ?? 'unknown')}
			</span>
		</div>

		<!-- MTTR -->
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-2">Avg. MTTR</div>
			<span class="text-lg font-mono text-foreground">{formatMTTR(investigation.mttr_seconds)}</span>
		</div>

		<!-- Agent -->
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-2">Agent</div>
			<div class="flex items-center space-x-1.5">
				<div class="w-2 h-2 rounded-full {investigation.agent_summary?.status === 'online' ? 'bg-emerald-400' : 'bg-red-400'}"></div>
				<span class="text-xs text-foreground truncate">{investigation.agent_summary?.name || '--'}</span>
			</div>
		</div>

		<!-- Previous Incidents -->
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-2">Previous</div>
			<span class="text-lg font-mono text-foreground">{investigation.previous_incidents?.length ?? 0}</span>
		</div>
	</div>

	<!-- Timeline -->
	{#if investigation.timeline && investigation.timeline.length > 0}
		<IncidentTimeline events={investigation.timeline} />
	{/if}

	<!-- Correlated Failures (Sibling Monitors) -->
	{#if investigation.sibling_monitors && investigation.sibling_monitors.length > 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="px-5 py-3.5 border-b border-border flex items-center space-x-2">
				<Server class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">Correlated Monitors</h3>
				<span class="text-xs text-muted-foreground">(same agent)</span>
			</div>
			<div class="divide-y divide-border/50">
				{#each investigation.sibling_monitors as sibling}
					<div class="px-5 py-3 flex items-center justify-between">
						<div class="flex items-center space-x-2.5">
							<div class="w-2 h-2 rounded-full {sibling.has_incident ? 'bg-red-400 animate-pulse' : sibling.status === 'up' ? 'bg-emerald-400' : 'bg-muted-foreground'}"></div>
							<a href="/monitors/{sibling.id}" class="text-xs font-medium text-foreground hover:text-accent transition-colors">
								{sibling.name}
							</a>
							<span class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground uppercase font-mono">{sibling.type}</span>
						</div>
						<div class="flex items-center space-x-2">
							<span class="text-[10px] font-mono text-muted-foreground truncate max-w-[140px]">{sibling.target}</span>
							{#if sibling.has_incident}
								<span class="text-[10px] px-1.5 py-0.5 rounded bg-red-500/10 text-red-400 border border-red-500/20">incident</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- System Context -->
	{#if investigation.system_metrics && investigation.system_metrics.length > 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="px-5 py-3.5 border-b border-border flex items-center space-x-2">
				<Cpu class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">System Context</h3>
			</div>
			<div class="divide-y divide-border/50">
				{#each investigation.system_metrics as metric}
					<div class="px-5 py-3 flex items-center justify-between">
						<div class="flex items-center space-x-2">
							<span class="text-xs text-foreground">{metric.monitor_name}</span>
							<span class="text-[10px] font-mono text-muted-foreground">{metric.target}</span>
						</div>
						<div class="flex items-center space-x-2">
							<span class="text-xs font-mono text-foreground">{metric.value || '--'}</span>
							<div class="w-2 h-2 rounded-full {metric.status === 'up' ? 'bg-emerald-400' : 'bg-red-400'}"></div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Cert Details -->
	{#if investigation.cert_details}
		<div class="bg-card border border-border rounded-lg">
			<div class="px-5 py-3.5 border-b border-border flex items-center space-x-2">
				<Shield class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">Certificate Details</h3>
			</div>
			<div class="p-5 grid grid-cols-2 gap-3">
				<div>
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-1">Issuer</div>
					<span class="text-xs text-foreground">{investigation.cert_details.issuer}</span>
				</div>
				<div>
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-1">Expires In</div>
					<span class="text-xs font-mono {(investigation.cert_details.expiry_days ?? 0) < 30 ? 'text-red-400' : 'text-foreground'}">
						{investigation.cert_details.expiry_days ?? '--'} days
					</span>
				</div>
				<div>
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-1">Algorithm</div>
					<span class="text-xs font-mono text-foreground">{investigation.cert_details.algorithm}</span>
				</div>
				<div>
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-1">Chain Valid</div>
					<span class="text-xs {investigation.cert_details.chain_valid ? 'text-emerald-400' : 'text-red-400'}">
						{investigation.cert_details.chain_valid ? 'Yes' : 'No'}
					</span>
				</div>
			</div>
		</div>
	{/if}

	<!-- Previous Incidents (History) -->
	{#if investigation.previous_incidents && investigation.previous_incidents.length > 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="px-5 py-3.5 border-b border-border flex items-center space-x-2">
				<History class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">Incident History</h3>
			</div>
			<div class="divide-y divide-border/50">
				{#each investigation.previous_incidents as prev}
					<div class="px-5 py-3">
						<div class="flex items-center justify-between">
							<div class="flex items-center space-x-2">
								<div class="w-2 h-2 rounded-full {prev.status === 'resolved' ? 'bg-emerald-400' : 'bg-red-400'}"></div>
								<span class="text-xs text-foreground">{formatTimeAgo(prev.started_at)}</span>
								<span class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground">{prev.status}</span>
							</div>
							{#if prev.ttr_seconds != null}
								<span class="text-[10px] font-mono text-muted-foreground">TTR: {formatMTTR(prev.ttr_seconds)}</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>

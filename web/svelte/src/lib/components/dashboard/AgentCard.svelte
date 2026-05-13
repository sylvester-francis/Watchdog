<script lang="ts">
	import { formatTimeAgo } from '$lib/utils';
	import type { Agent, DashboardStats } from '$lib/types';
	import { Button, SectionHeader, StatusPip } from '@sylvester-francis/watchdog-ui';

	interface Props {
		agents: Agent[];
		stats: DashboardStats;
		onCreateAgent: () => void;
		canWrite?: boolean;
	}

	let { agents, stats, onCreateAgent, canWrite = true }: Props = $props();
</script>

<section>
	<SectionHeader title="Agents">
		{#snippet meta()}
			{stats.online_agents}/{stats.total_agents} online
		{/snippet}
		{#snippet action()}
			{#if canWrite}
				<Button variant="primary" size="sm" onclick={onCreateAgent}>New Agent</Button>
			{/if}
		{/snippet}
	</SectionHeader>

	{#if agents.length > 0}
		<div class="divide-y divide-border/40">
			{#each agents as agent (agent.id)}
				<div class="flex items-center justify-between gap-3 py-3 transition-colors hover:bg-muted/30">
					<div class="min-w-0">
						<div class="flex items-center gap-2">
							<StatusPip tone={agent.status === 'online' ? 'success' : 'muted'} />
							<p class="truncate text-sm text-foreground">{agent.name}</p>
						</div>
						<p class="mt-0.5 ml-3.5 font-mono tabular-nums text-[11px] text-muted-foreground">
							{agent.last_seen_at ? formatTimeAgo(agent.last_seen_at) : 'Never connected'}
						</p>
					</div>
					<span class="shrink-0 font-mono tabular-nums text-[11px] uppercase tracking-wider {agent.status === 'online' ? 'text-success' : 'text-muted-foreground'}">
						{agent.status}
					</span>
				</div>
			{/each}
		</div>
	{:else}
		<div class="pt-5">
			<p class="mb-4 text-sm font-medium text-foreground">Get started in 3 steps</p>
			<ol class="space-y-3 font-mono tabular-nums text-xs text-muted-foreground">
				<li class="flex items-start gap-3">
					<span class="mt-0.5 shrink-0 text-foreground/40">01</span>
					<div class="font-sans">
						<p class="text-xs font-medium text-foreground">Create an agent</p>
						<p class="mt-0.5 text-[11px] text-muted-foreground">Click <strong class="text-foreground">New Agent</strong> above to register an agent and get an API key.</p>
					</div>
				</li>
				<li class="flex items-start gap-3">
					<span class="mt-0.5 shrink-0 text-foreground/40">02</span>
					<div class="font-sans">
						<p class="text-xs font-medium text-foreground">Install the agent</p>
						<code class="mt-1 block select-all break-all border border-border bg-background px-2 py-1.5 font-mono text-[11px] text-foreground">curl -sSL https://{window.location.host}/install | sh -s -- --api-key YOUR_KEY</code>
					</div>
				</li>
				<li class="flex items-start gap-3">
					<span class="mt-0.5 shrink-0 text-foreground/40">03</span>
					<div class="font-sans">
						<p class="text-xs font-medium text-foreground">Create monitors</p>
						<p class="mt-0.5 text-[11px] text-muted-foreground">Add HTTP, TCP, Ping, or TLS monitors to track your services.</p>
					</div>
				</li>
			</ol>
		</div>
	{/if}
</section>

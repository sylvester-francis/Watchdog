<script lang="ts">
	import { Server } from 'lucide-svelte';
	import { formatTimeAgo } from '$lib/utils';
	import type { Agent, DashboardStats } from '$lib/types';
	import Button from '$lib/ui/Button.svelte';
	import Pill from '$lib/ui/Pill.svelte';

	interface Props {
		agents: Agent[];
		stats: DashboardStats;
		onCreateAgent: () => void;
	}

	let { agents, stats, onCreateAgent }: Props = $props();
</script>

<div class="bg-card border border-border rounded-lg self-start">
	<div class="px-4 py-3 border-b border-border flex items-center justify-between">
		<div class="flex items-center space-x-2">
			<h2 class="text-sm font-medium text-foreground">Agents</h2>
			<div class="flex items-center space-x-1.5">
				<div class="w-1.5 h-1.5 rounded-full {stats.online_agents > 0 ? 'bg-emerald-400' : 'bg-muted-foreground'}"></div>
				<span class="text-[10px] text-muted-foreground font-mono">{stats.online_agents}/{stats.total_agents} online</span>
			</div>
		</div>
		<Button variant="primary" size="sm" onclick={onCreateAgent}>New Agent</Button>
	</div>

	{#if agents.length > 0}
		<div class="divide-y divide-border/30">
			{#each agents as agent (agent.id)}
				<div class="flex items-center justify-between px-4 py-2.5 transition-colors hover:bg-card-elevated">
					<div class="flex items-center space-x-3">
						<div class="w-7 h-7 rounded-md {agent.status === 'online' ? 'bg-emerald-500/10' : 'bg-muted/50'} flex items-center justify-center">
							<Server class="w-3.5 h-3.5 {agent.status === 'online' ? 'text-emerald-400' : 'text-muted-foreground'}" />
						</div>
						<div>
							<p class="text-sm font-medium text-foreground">{agent.name}</p>
							<p class="text-[10px] text-muted-foreground">
								{agent.last_seen_at ? formatTimeAgo(agent.last_seen_at) : 'Never connected'}
							</p>
						</div>
					</div>
					<Pill tone={agent.status === 'online' ? 'up' : 'neutral'}>
						{#if agent.status === 'online'}
							<span class="w-1.5 h-1.5 rounded-full bg-status-up animate-pulse inline-block mr-1"></span>
						{/if}
						{agent.status}
					</Pill>
				</div>
			{/each}
		</div>
	{:else}
		<div class="px-4 py-6">
			<p class="text-sm font-medium text-foreground mb-4">Get started in 3 steps</p>
			<div class="space-y-3">
				<div class="flex items-start space-x-3">
					<span class="shrink-0 w-5 h-5 rounded-full bg-accent/15 text-accent text-[10px] font-bold flex items-center justify-center mt-0.5">1</span>
					<div>
						<p class="text-xs font-medium text-foreground">Create an agent</p>
						<p class="text-[10px] text-muted-foreground mt-0.5">Click <strong>New Agent</strong> above to register an agent and get an API key.</p>
					</div>
				</div>
				<div class="flex items-start space-x-3">
					<span class="shrink-0 w-5 h-5 rounded-full bg-accent/15 text-accent text-[10px] font-bold flex items-center justify-center mt-0.5">2</span>
					<div>
						<p class="text-xs font-medium text-foreground">Install the agent</p>
						<p class="text-[10px] text-muted-foreground mt-0.5 font-mono bg-muted/50 rounded px-1.5 py-1 mt-1 select-all">curl -fsSL https://usewatchdog.dev/install | bash</p>
					</div>
				</div>
				<div class="flex items-start space-x-3">
					<span class="shrink-0 w-5 h-5 rounded-full bg-accent/15 text-accent text-[10px] font-bold flex items-center justify-center mt-0.5">3</span>
					<div>
						<p class="text-xs font-medium text-foreground">Create monitors</p>
						<p class="text-[10px] text-muted-foreground mt-0.5">Add HTTP, TCP, Ping, or TLS monitors to track your services.</p>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>

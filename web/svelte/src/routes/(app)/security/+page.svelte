<script lang="ts">
	import { onMount } from 'svelte';
	import { ShieldAlert, ShieldCheck, ShieldX, UserX, RefreshCw } from 'lucide-svelte';
	import { system as systemApi } from '$lib/api';
	import type { SecurityEvent } from '$lib/types';

	let events = $state<SecurityEvent[]>([]);
	let loading = $state(true);
	let error = $state('');

	async function loadEvents() {
		loading = true;
		error = '';
		try {
			const res = await systemApi.getSecurityEvents();
			events = res.data;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load security events';
		} finally {
			loading = false;
		}
	}

	onMount(loadEvents);

	function actionLabel(action: string): string {
		switch (action) {
			case 'register_success': return 'Registration';
			case 'register_blocked': return 'Blocked';
			case 'login_failed': return 'Login Failed';
			default: return action;
		}
	}

	function actionColor(action: string): string {
		switch (action) {
			case 'register_success': return 'text-emerald-400 bg-emerald-500/10';
			case 'register_blocked': return 'text-red-400 bg-red-500/10';
			case 'login_failed': return 'text-yellow-400 bg-yellow-500/10';
			default: return 'text-muted-foreground bg-muted/50';
		}
	}

	function reasonLabel(metadata: Record<string, string>): string {
		const reason = metadata?.reason;
		if (!reason) return '';
		switch (reason) {
			case 'honeypot': return 'Bot (honeypot)';
			case 'blocked_domain': return 'Disposable email';
			default: return reason;
		}
	}

	function timeAgo(dateStr: string): string {
		const now = Date.now();
		const then = new Date(dateStr).getTime();
		const diffSec = Math.floor((now - then) / 1000);

		if (diffSec < 60) return `${diffSec}s ago`;
		if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
		if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
		return `${Math.floor(diffSec / 86400)}d ago`;
	}

	const blockedCount = $derived(events.filter(e => e.action === 'register_blocked').length);
	const registrations = $derived(events.filter(e => e.action === 'register_success').length);
	const failedLogins = $derived(events.filter(e => e.action === 'login_failed').length);
</script>

<svelte:head>
	<title>Security Events - WatchDog</title>
</svelte:head>

<div class="max-w-6xl mx-auto space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-lg font-semibold text-foreground">Security Events</h1>
			<p class="text-xs text-muted-foreground mt-0.5">Registration attempts, blocked bots, and failed logins.</p>
		</div>
		<button onclick={loadEvents} disabled={loading}
			class="flex items-center space-x-1.5 px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground bg-card border border-border/50 rounded-md transition-colors disabled:opacity-50">
			<RefreshCw class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" />
			<span>Refresh</span>
		</button>
	</div>

	<!-- Stats -->
	<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
		<div class="bg-card border border-border/50 rounded-lg p-4 flex items-center space-x-3">
			<div class="w-9 h-9 bg-emerald-500/10 rounded-md flex items-center justify-center">
				<ShieldCheck class="w-5 h-5 text-emerald-400" />
			</div>
			<div>
				<p class="text-xl font-semibold text-foreground">{registrations}</p>
				<p class="text-[11px] text-muted-foreground">Registrations</p>
			</div>
		</div>
		<div class="bg-card border border-border/50 rounded-lg p-4 flex items-center space-x-3">
			<div class="w-9 h-9 bg-red-500/10 rounded-md flex items-center justify-center">
				<ShieldX class="w-5 h-5 text-red-400" />
			</div>
			<div>
				<p class="text-xl font-semibold text-foreground">{blockedCount}</p>
				<p class="text-[11px] text-muted-foreground">Blocked Attempts</p>
			</div>
		</div>
		<div class="bg-card border border-border/50 rounded-lg p-4 flex items-center space-x-3">
			<div class="w-9 h-9 bg-yellow-500/10 rounded-md flex items-center justify-center">
				<UserX class="w-5 h-5 text-yellow-400" />
			</div>
			<div>
				<p class="text-xl font-semibold text-foreground">{failedLogins}</p>
				<p class="text-[11px] text-muted-foreground">Failed Logins</p>
			</div>
		</div>
	</div>

	{#if error}
		<div class="bg-destructive/10 border border-destructive/20 rounded-md px-4 py-3">
			<p class="text-sm text-destructive">{error}</p>
		</div>
	{/if}

	<!-- Events Table -->
	<div class="bg-card border border-border/50 rounded-lg overflow-hidden">
		<div class="px-4 py-3 border-b border-border/50">
			<h2 class="text-sm font-medium text-foreground flex items-center space-x-2">
				<ShieldAlert class="w-4 h-4 text-muted-foreground" />
				<span>Recent Events ({events.length})</span>
			</h2>
		</div>

		{#if loading}
			<div class="p-8 text-center text-sm text-muted-foreground">Loading security events...</div>
		{:else if events.length === 0}
			<div class="p-8 text-center text-sm text-muted-foreground">No security events found.</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border/50 text-left">
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Time</th>
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Action</th>
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Email</th>
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">IP Address</th>
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Reason</th>
						</tr>
					</thead>
					<tbody>
						{#each events as event}
							<tr class="border-b border-border/30 hover:bg-muted/30 transition-colors">
								<td class="px-4 py-2.5 text-xs text-muted-foreground whitespace-nowrap" title={event.created_at}>
									{timeAgo(event.created_at)}
								</td>
								<td class="px-4 py-2.5">
									<span class="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium {actionColor(event.action)}">
										{actionLabel(event.action)}
									</span>
								</td>
								<td class="px-4 py-2.5 text-xs text-foreground font-mono">
									{event.metadata?.email || event.user_email || '-'}
								</td>
								<td class="px-4 py-2.5 text-xs text-muted-foreground font-mono">
									{event.ip_address || '-'}
								</td>
								<td class="px-4 py-2.5 text-xs text-muted-foreground">
									{reasonLabel(event.metadata)}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

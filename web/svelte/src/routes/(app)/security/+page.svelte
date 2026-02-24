<script lang="ts">
	import { onMount } from 'svelte';
	import { ShieldAlert, ShieldCheck, ShieldX, UserX, RefreshCw, Trash2 } from 'lucide-svelte';
	import { system as systemApi } from '$lib/api';
	import type { SecurityEvent } from '$lib/types';

	let events = $state<SecurityEvent[]>([]);
	let loading = $state(true);
	let error = $state('');
	let deletingUser = $state<string | null>(null);
	let deleteConfirm = $state<string | null>(null);

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

	// Extract unique registered users from register_success events for the users list
	const registeredUsers = $derived(
		events
			.filter(e => e.action === 'register_success' && e.metadata?.email)
			.reduce((acc, e) => {
				const email = e.metadata.email;
				if (!acc.some(u => u.email === email)) {
					acc.push({ email, ip: e.ip_address, time: e.created_at, id: e.metadata?.user_id });
				}
				return acc;
			}, [] as { email: string; ip: string; time: string; id?: string }[])
	);

	async function deleteUser(userId: string, email: string) {
		if (deleteConfirm !== userId) {
			deleteConfirm = userId;
			return;
		}
		deletingUser = userId;
		error = '';
		try {
			await systemApi.deleteUser(userId);
			deleteConfirm = null;
			await loadEvents();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete user';
		} finally {
			deletingUser = null;
		}
	}

	function cancelDelete() {
		deleteConfirm = null;
	}
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

	<!-- Registered Users -->
	{#if registeredUsers.length > 0}
		<div class="bg-card border border-border/50 rounded-lg overflow-hidden">
			<div class="px-4 py-3 border-b border-border/50">
				<h2 class="text-sm font-medium text-foreground flex items-center space-x-2">
					<UserX class="w-4 h-4 text-muted-foreground" />
					<span>Registered Users ({registeredUsers.length})</span>
				</h2>
			</div>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border/50 text-left">
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Email</th>
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">IP Address</th>
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Registered</th>
							<th class="px-4 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each registeredUsers as user}
							<tr class="border-b border-border/30 hover:bg-muted/30 transition-colors">
								<td class="px-4 py-2.5 text-xs text-foreground font-mono">{user.email}</td>
								<td class="px-4 py-2.5 text-xs text-muted-foreground font-mono">{user.ip || '-'}</td>
								<td class="px-4 py-2.5 text-xs text-muted-foreground" title={user.time}>{timeAgo(user.time)}</td>
								<td class="px-4 py-2.5 text-right">
									{#if user.id}
										{#if deleteConfirm === user.id}
											<span class="text-[11px] text-destructive mr-2">Delete this user?</span>
											<button onclick={() => deleteUser(user.id!, user.email)}
												disabled={deletingUser === user.id}
												class="text-[11px] px-2 py-0.5 rounded bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors disabled:opacity-50 mr-1">
												{deletingUser === user.id ? 'Deleting...' : 'Confirm'}
											</button>
											<button onclick={cancelDelete}
												class="text-[11px] px-2 py-0.5 rounded bg-muted text-muted-foreground hover:text-foreground transition-colors">
												Cancel
											</button>
										{:else}
											<button onclick={() => deleteUser(user.id!, user.email)}
												class="inline-flex items-center space-x-1 text-[11px] px-2 py-0.5 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors">
												<Trash2 class="w-3 h-3" />
												<span>Delete</span>
											</button>
										{/if}
									{:else}
										<span class="text-[11px] text-muted-foreground/50">-</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
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

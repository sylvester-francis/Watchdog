<script lang="ts">
	import { onMount } from 'svelte';
	import { RefreshCw, Trash2 } from 'lucide-svelte';
	import { Alert } from '@sylvester-francis/watchdog-ui';
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

	function actionTextClass(action: string): string {
		switch (action) {
			case 'register_success': return 'text-success';
			case 'register_blocked': return 'text-destructive';
			case 'login_failed': return 'text-warning';
			default: return 'text-muted-foreground';
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

	const blockedCount = $derived(events.filter((e) => e.action === 'register_blocked').length);
	const registrations = $derived(events.filter((e) => e.action === 'register_success').length);
	const failedLogins = $derived(events.filter((e) => e.action === 'login_failed').length);

	const registeredUsers = $derived(
		events
			.filter((e) => e.action === 'register_success' && e.metadata?.email)
			.reduce(
				(acc, e) => {
					const email = e.metadata.email;
					if (!acc.some((u) => u.email === email)) {
						acc.push({ email, ip: e.ip_address, time: e.created_at, id: e.metadata?.user_id });
					}
					return acc;
				},
				[] as { email: string; ip: string; time: string; id?: string }[]
			)
	);

	async function deleteUser(userId: string, _email: string) {
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

<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-8 sm:px-6 sm:py-10">
	<!-- Header -->
	<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-4">
		<div class="min-w-0">
			<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
				<span class="uppercase tracking-wider">Security</span>
			</div>
			<h1 class="mt-1.5 text-2xl font-medium text-foreground sm:text-3xl">Security Events</h1>
			<p class="mt-1 text-sm text-muted-foreground">Registration attempts, blocked bots, and failed logins.</p>
		</div>
		<button
			onclick={loadEvents}
			disabled={loading}
			class="flex items-center gap-1 self-start text-sm text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline disabled:opacity-50 sm:self-auto"
		>
			<RefreshCw class="h-3.5 w-3.5 {loading ? 'animate-spin' : ''}" />
			<span>Refresh</span>
		</button>
	</header>

	<!-- Stats — hairline columns -->
	<div class="mt-8 grid grid-cols-1 gap-px overflow-hidden border-y border-border bg-border sm:grid-cols-3">
		<div class="flex flex-col bg-background px-4 py-3.5">
			<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Registrations</div>
			<div class="mt-1 font-mono tabular-nums text-lg text-foreground">{registrations}</div>
		</div>
		<div class="flex flex-col bg-background px-4 py-3.5">
			<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Blocked Attempts</div>
			<div class="mt-1 font-mono tabular-nums text-lg {blockedCount > 0 ? 'text-destructive' : 'text-foreground'}">{blockedCount}</div>
		</div>
		<div class="flex flex-col bg-background px-4 py-3.5">
			<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Failed Logins</div>
			<div class="mt-1 font-mono tabular-nums text-lg {failedLogins > 0 ? 'text-warning' : 'text-foreground'}">{failedLogins}</div>
		</div>
	</div>

	{#if error}
		<div class="mt-6">
			<Alert tone="down">{error}</Alert>
		</div>
	{/if}

	<!-- Registered Users -->
	{#if registeredUsers.length > 0}
		<section class="mt-10">
			<div class="flex items-baseline gap-2 border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Registered Users</h3>
				<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{registeredUsers.length}</span>
			</div>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border">
							<th class="py-2.5 pl-1 pr-4 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Email</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">IP Address</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Registered</th>
							<th class="px-4 py-2.5 text-right text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/40">
						{#each registeredUsers as user}
							<tr class="transition-colors hover:bg-muted/30">
								<td class="py-3 pl-1 pr-4 font-mono tabular-nums text-xs text-foreground">{user.email}</td>
								<td class="px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground">{user.ip || '—'}</td>
								<td class="px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground" title={user.time}>{timeAgo(user.time)}</td>
								<td class="px-4 py-3 text-right text-xs">
									{#if user.id}
										{#if deleteConfirm === user.id}
											<span class="mr-3 text-destructive">Delete this user?</span>
											<button
												onclick={() => deleteUser(user.id!, user.email)}
												disabled={deletingUser === user.id}
												class="mr-3 text-destructive underline-offset-4 transition-colors hover:underline disabled:opacity-50"
											>
												{deletingUser === user.id ? 'Deleting…' : 'Confirm'}
											</button>
											<button
												onclick={cancelDelete}
												class="text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
											>
												Cancel
											</button>
										{:else}
											<button
												onclick={() => deleteUser(user.id!, user.email)}
												class="inline-flex items-center gap-1 text-destructive underline-offset-4 transition-colors hover:underline"
											>
												<Trash2 class="h-3 w-3" />
												<span>Delete</span>
											</button>
										{/if}
									{:else}
										<span class="text-muted-foreground/50">—</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</section>
	{/if}

	<!-- Events Table -->
	<section class="mt-10">
		<div class="flex items-baseline gap-2 border-b border-border pb-3">
			<h3 class="text-sm font-medium text-foreground">Recent Events</h3>
			<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{events.length}</span>
		</div>

		{#if loading}
			<p class="pt-4 text-sm text-muted-foreground">Loading security events…</p>
		{:else if events.length === 0}
			<p class="pt-4 text-xs text-muted-foreground">
				No security events. Registration attempts, blocked bots, and failed logins will appear here.
			</p>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border">
							<th class="py-2.5 pl-1 pr-4 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Time</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Action</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Email</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">IP Address</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Reason</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/40">
						{#each events as event}
							<tr class="transition-colors hover:bg-muted/30">
								<td class="whitespace-nowrap py-3 pl-1 pr-4 font-mono tabular-nums text-xs text-muted-foreground" title={event.created_at}>
									{timeAgo(event.created_at)}
								</td>
								<td class="whitespace-nowrap px-4 py-3 font-mono tabular-nums text-[11px] uppercase tracking-wider {actionTextClass(event.action)}">
									{actionLabel(event.action)}
								</td>
								<td class="px-4 py-3 font-mono tabular-nums text-xs text-foreground">
									{event.metadata?.email || event.user_email || '—'}
								</td>
								<td class="px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground">
									{event.ip_address || '—'}
								</td>
								<td class="px-4 py-3 text-xs text-muted-foreground">{reasonLabel(event.metadata)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</div>

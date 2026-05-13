<script lang="ts">
	import { onMount } from 'svelte';
	import { ChevronLeft, ChevronRight, Download, Search, Loader2 } from 'lucide-svelte';
	import { PageHero } from '@sylvester-francis/watchdog-ui';
	import { system as systemApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { AuditLogEntry, PaginationMeta } from '$lib/types';

	const toast = getToasts();

	type CategoryTab = 'all' | 'auth' | 'monitor' | 'agent' | 'system';

	const categoryActions: Record<CategoryTab, string[]> = {
		all: [],
		auth: ['login_success', 'login_failed', 'register_success', 'register_blocked', 'logout', 'password_changed', 'password_reset_by_admin'],
		monitor: ['monitor_created', 'monitor_updated', 'monitor_deleted', 'incident_acknowledged', 'incident_resolved'],
		agent: ['agent_created', 'agent_deleted', 'maintenance_window_created', 'maintenance_window_updated', 'maintenance_window_deleted'],
		system: ['api_token_created', 'api_token_revoked', 'channel_created', 'channel_deleted', 'settings_changed', 'user_deleted'],
	};

	const tabs: { value: CategoryTab; label: string }[] = [
		{ value: 'all', label: 'All' },
		{ value: 'auth', label: 'Auth' },
		{ value: 'monitor', label: 'Monitor' },
		{ value: 'agent', label: 'Agent' },
		{ value: 'system', label: 'System' },
	];

	let logs = $state<AuditLogEntry[]>([]);
	let meta = $state<PaginationMeta>({ page: 1, per_page: 25, total: 0, pages: 0 });
	let loading = $state(true);
	let activeTab = $state<CategoryTab>('all');
	let searchQuery = $state('');
	let dateFrom = $state('');
	let dateTo = $state('');
	let selectedAction = $state('');

	let filteredLogs = $derived(
		searchQuery
			? logs.filter(l =>
				l.action.includes(searchQuery.toLowerCase()) ||
				l.user_email.toLowerCase().includes(searchQuery.toLowerCase()) ||
				l.ip_address.includes(searchQuery)
			)
			: logs
	);

	function actionTextClass(action: string): string {
		if (action === 'login_success' || action === 'register_success') return 'text-success';
		if (action === 'login_failed' || action === 'register_blocked') return 'text-destructive';
		if (action.endsWith('_deleted') || action.endsWith('_revoked')) return 'text-destructive';
		if (action.endsWith('_updated') || action === 'password_changed' || action === 'settings_changed') return 'text-warning';
		if (action.includes('acknowledged')) return 'text-warning';
		if (action.includes('resolved')) return 'text-success';
		return 'text-muted-foreground';
	}

	function formatDate(iso: string): string {
		const d = new Date(iso);
		return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function formatTime(iso: string): string {
		const d = new Date(iso);
		return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' });
	}

	async function loadData(page = 1) {
		loading = true;
		try {
			const res = await systemApi.getAuditLogs({
				page,
				per_page: 25,
				action: selectedAction || undefined,
				from: dateFrom || undefined,
				to: dateTo || undefined,
			});
			logs = res.data ?? [];
			meta = res.meta;
		} catch {
			toast.error('Failed to load audit logs');
		} finally {
			loading = false;
		}
	}

	function handleTabChange(tab: CategoryTab) {
		activeTab = tab;
		selectedAction = '';
		loadData(1);
	}

	function handleActionFilter(action: string) {
		selectedAction = action;
		loadData(1);
	}

	function handleDateFilter() {
		loadData(1);
	}

	function goToPage(page: number) {
		if (page >= 1 && page <= meta.pages) {
			loadData(page);
		}
	}

	function exportCSV() {
		const headers = ['Timestamp', 'Action', 'User', 'IP Address', 'Metadata'];
		const rows = filteredLogs.map(l => [
			l.created_at,
			l.action,
			l.user_email,
			l.ip_address,
			JSON.stringify(l.metadata),
		]);

		const csv = [headers, ...rows].map(r => r.map(c => `"${String(c).replace(/"/g, '""')}"`).join(',')).join('\n');
		const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.csv`;
		a.click();
		URL.revokeObjectURL(url);
	}

	function pageNumbers(): number[] {
		const pages: number[] = [];
		const start = Math.max(1, meta.page - 2);
		const end = Math.min(meta.pages, meta.page + 2);
		for (let i = start; i <= end; i++) {
			pages.push(i);
		}
		return pages;
	}

	onMount(() => {
		loadData();
	});

	const inputClass =
		'border border-border bg-background px-2.5 py-1.5 text-xs text-foreground placeholder:text-muted-foreground/50 focus:border-foreground/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-foreground/30';
</script>

<svelte:head>
	<title>Audit Log - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
	<PageHero
		meta="Audit"
		title="{meta.total} event{meta.total !== 1 ? 's' : ''} total"
	>
		{#snippet action()}
			<button
				onclick={exportCSV}
				class="flex items-center gap-1 self-start text-sm text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline sm:self-auto"
			>
				<Download class="h-3.5 w-3.5" />
				<span>Export CSV</span>
			</button>
		{/snippet}
	</PageHero>

	<!-- Category tabs -->
	<div class="mt-8 flex flex-wrap items-center gap-1 font-mono tabular-nums text-xs">
		{#each tabs as tab}
			<button
				onclick={() => handleTabChange(tab.value)}
				class="px-2.5 py-1 transition-colors {activeTab === tab.value
					? 'font-medium text-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<!-- Filters row -->
	<div class="mt-3 flex flex-wrap items-center gap-2">
		{#if categoryActions[activeTab].length > 0}
			<select
				bind:value={selectedAction}
				onchange={() => handleActionFilter(selectedAction)}
				class="{inputClass} cursor-pointer"
			>
				<option value="">All {activeTab} actions</option>
				{#each categoryActions[activeTab] as action}
					<option value={action}>{action.replace(/_/g, ' ')}</option>
				{/each}
			</select>
		{/if}

		<input type="date" bind:value={dateFrom} onchange={handleDateFilter} class={inputClass} placeholder="From" />
		<input type="date" bind:value={dateTo} onchange={handleDateFilter} class={inputClass} placeholder="To" />

		<div class="relative min-w-[160px] max-w-[240px] flex-1">
			<Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground/50" />
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search results..."
				class="{inputClass} w-full pl-8"
			/>
		</div>
	</div>

	<!-- Table -->
	<div class="mt-6">
		{#if loading}
			<div class="flex items-center justify-center py-16">
				<Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
			</div>
		{:else if filteredLogs.length === 0}
			<section>
				<div class="border-b border-border pb-3">
					<h3 class="text-sm font-medium text-foreground">No audit logs found</h3>
				</div>
				<p class="pt-4 text-xs text-muted-foreground">Try adjusting your filters or date range.</p>
			</section>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full">
					<thead>
						<tr class="border-b border-border">
							<th class="py-2.5 pl-1 pr-4 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Time</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Action</th>
							<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground md:table-cell">User</th>
							<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground lg:table-cell">IP Address</th>
							<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground xl:table-cell">Details</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/40">
						{#each filteredLogs as log}
							<tr class="transition-colors hover:bg-muted/30">
								<td class="py-3 pl-1 pr-4 font-mono tabular-nums">
									<div class="text-xs text-foreground">{formatDate(log.created_at)}</div>
									<div class="text-[10px] text-muted-foreground">{formatTime(log.created_at)}</div>
								</td>
								<td class="whitespace-nowrap px-4 py-3 font-mono tabular-nums text-[11px] {actionTextClass(log.action)}">
									{log.action.replace(/_/g, ' ')}
								</td>
								<td class="hidden px-4 py-3 font-mono tabular-nums text-xs text-foreground md:table-cell">
									{log.user_email || '—'}
								</td>
								<td class="hidden px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground lg:table-cell">
									{log.ip_address || '—'}
								</td>
								<td class="hidden px-4 py-3 xl:table-cell">
									{#if log.metadata && Object.keys(log.metadata).length > 0}
										<div class="flex flex-wrap gap-2 font-mono tabular-nums text-[10px] text-muted-foreground">
											{#each Object.entries(log.metadata).slice(0, 3) as [key, value]}
												<span class="truncate max-w-[140px]" title="{key}: {value}">
													{key}: {value}
												</span>
											{/each}
										</div>
									{:else}
										<span class="text-xs text-muted-foreground/40">—</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Pagination -->
			{#if meta.pages > 1}
				<div class="mt-4 flex items-center justify-between">
					<p class="font-mono tabular-nums text-xs text-muted-foreground">
						Page {meta.page} of {meta.pages} ({meta.total} total)
					</p>
					<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
						<button
							onclick={() => goToPage(meta.page - 1)}
							disabled={meta.page <= 1}
							class="text-muted-foreground transition-colors hover:text-foreground disabled:opacity-30"
						>
							<ChevronLeft class="h-3.5 w-3.5" />
						</button>
						{#each pageNumbers() as page}
							<button
								onclick={() => goToPage(page)}
								class="px-1 transition-colors {page === meta.page
									? 'font-medium text-foreground'
									: 'text-muted-foreground hover:text-foreground'}"
							>
								{page}
							</button>
						{/each}
						<button
							onclick={() => goToPage(meta.page + 1)}
							disabled={meta.page >= meta.pages}
							class="text-muted-foreground transition-colors hover:text-foreground disabled:opacity-30"
						>
							<ChevronRight class="h-3.5 w-3.5" />
						</button>
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>

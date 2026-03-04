<script lang="ts">
	import { onMount } from 'svelte';
	import { ScrollText, ChevronLeft, ChevronRight, Download, Search, Loader2 } from 'lucide-svelte';
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

	function actionBadgeClass(action: string): string {
		if (action === 'login_success' || action === 'register_success') return 'bg-green-500/15 text-green-400';
		if (action === 'login_failed' || action === 'register_blocked') return 'bg-red-500/15 text-red-400';
		if (action.endsWith('_created')) return 'bg-blue-500/15 text-blue-400';
		if (action.endsWith('_updated') || action === 'password_changed' || action === 'settings_changed') return 'bg-yellow-500/15 text-yellow-400';
		if (action.endsWith('_deleted') || action.endsWith('_revoked')) return 'bg-red-500/15 text-red-400';
		if (action.includes('acknowledged')) return 'bg-yellow-500/15 text-yellow-400';
		if (action.includes('resolved')) return 'bg-emerald-500/15 text-emerald-400';
		if (action.includes('maintenance')) return 'bg-purple-500/15 text-purple-400';
		if (action === 'logout') return 'bg-muted/50 text-muted-foreground';
		return 'bg-muted/50 text-muted-foreground';
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
		// Reset action filter to first action in category, or empty for 'all'
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
</script>

<svelte:head>
	<title>Audit Log - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up">
	<!-- Header -->
	<div class="flex items-center justify-between mb-5">
		<div>
			<h1 class="text-lg font-semibold text-foreground">Audit Log</h1>
			<p class="text-xs text-muted-foreground mt-0.5">
				{meta.total} event{meta.total !== 1 ? 's' : ''} total
			</p>
		</div>
		<button
			onclick={exportCSV}
			class="flex items-center space-x-1.5 px-3 py-1.5 bg-card border border-border rounded-md text-xs text-muted-foreground hover:text-foreground transition-colors"
		>
			<Download class="w-3.5 h-3.5" />
			<span>Export CSV</span>
		</button>
	</div>

	<!-- Category tabs -->
	<div class="flex items-center gap-1 mb-4">
		{#each tabs as tab}
			<button
				onclick={() => handleTabChange(tab.value)}
				class="px-2.5 py-1 text-xs rounded-md transition-colors {activeTab === tab.value
					? 'bg-foreground/[0.08] text-foreground font-medium'
					: 'text-muted-foreground hover:text-foreground hover:bg-foreground/[0.04]'}"
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<!-- Filters row -->
	<div class="flex flex-wrap items-center gap-2 mb-4">
		<!-- Action filter dropdown (shows actions from current category) -->
		{#if categoryActions[activeTab].length > 0}
			<select
				bind:value={selectedAction}
				onchange={() => handleActionFilter(selectedAction)}
				class="px-2.5 py-1.5 bg-card border border-border rounded-md text-xs text-foreground appearance-none cursor-pointer"
			>
				<option value="">All {activeTab} actions</option>
				{#each categoryActions[activeTab] as action}
					<option value={action}>{action.replace(/_/g, ' ')}</option>
				{/each}
			</select>
		{/if}

		<!-- Date range -->
		<input
			type="date"
			bind:value={dateFrom}
			onchange={handleDateFilter}
			class="px-2.5 py-1.5 bg-card border border-border rounded-md text-xs text-foreground"
			placeholder="From"
		/>
		<input
			type="date"
			bind:value={dateTo}
			onchange={handleDateFilter}
			class="px-2.5 py-1.5 bg-card border border-border rounded-md text-xs text-foreground"
			placeholder="To"
		/>

		<!-- Client-side search -->
		<div class="relative flex-1 min-w-[160px] max-w-[240px]">
			<Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search results..."
				class="w-full pl-8 pr-3 py-1.5 bg-card border border-border rounded-md text-xs text-foreground placeholder:text-muted-foreground"
			/>
		</div>
	</div>

	<!-- Table -->
	{#if loading}
		<div class="bg-card border border-border rounded-lg">
			<div class="flex items-center justify-center py-16">
				<Loader2 class="w-5 h-5 text-muted-foreground animate-spin" />
			</div>
		</div>
	{:else if filteredLogs.length === 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="p-12 text-center">
				<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
					<ScrollText class="w-6 h-6 text-muted-foreground/40" />
				</div>
				<p class="text-sm font-medium text-foreground mb-1">No audit logs found</p>
				<p class="text-xs text-muted-foreground">Try adjusting your filters or date range.</p>
			</div>
		</div>
	{:else}
		<div class="bg-card border border-border rounded-lg overflow-x-auto">
			<table class="w-full">
				<thead>
					<tr class="border-b border-border">
						<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Time</th>
						<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Action</th>
						<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">User</th>
						<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden lg:table-cell">IP Address</th>
						<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden xl:table-cell">Details</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border/50">
					{#each filteredLogs as log}
						<tr class="hover:bg-card-elevated transition-colors">
							<td class="px-4 py-3">
								<div class="text-xs text-foreground">{formatDate(log.created_at)}</div>
								<div class="text-[10px] font-mono text-muted-foreground">{formatTime(log.created_at)}</div>
							</td>
							<td class="px-4 py-3">
								<span class="px-2 py-0.5 text-[10px] font-medium rounded whitespace-nowrap {actionBadgeClass(log.action)}">
									{log.action.replace(/_/g, ' ')}
								</span>
							</td>
							<td class="px-4 py-3 hidden md:table-cell">
								<span class="text-xs text-foreground">{log.user_email || '--'}</span>
							</td>
							<td class="px-4 py-3 hidden lg:table-cell">
								<span class="text-xs font-mono text-muted-foreground">{log.ip_address || '--'}</span>
							</td>
							<td class="px-4 py-3 hidden xl:table-cell">
								{#if log.metadata && Object.keys(log.metadata).length > 0}
									<div class="flex flex-wrap gap-1">
										{#each Object.entries(log.metadata).slice(0, 3) as [key, value]}
											<span class="text-[10px] px-1.5 py-0.5 rounded bg-muted/50 text-muted-foreground font-mono truncate max-w-[140px]" title="{key}: {value}">
												{key}: {value}
											</span>
										{/each}
									</div>
								{:else}
									<span class="text-xs text-muted-foreground">--</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- Pagination -->
		{#if meta.pages > 1}
			<div class="flex items-center justify-between mt-4">
				<p class="text-xs text-muted-foreground">
					Page {meta.page} of {meta.pages} ({meta.total} total)
				</p>
				<div class="flex items-center space-x-1">
					<button
						onclick={() => goToPage(meta.page - 1)}
						disabled={meta.page <= 1}
						class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
					>
						<ChevronLeft class="w-4 h-4" />
					</button>
					{#each pageNumbers() as page}
						<button
							onclick={() => goToPage(page)}
							class="px-2.5 py-1 text-xs rounded-md transition-colors {page === meta.page
								? 'bg-foreground/[0.08] text-foreground font-medium'
								: 'text-muted-foreground hover:text-foreground hover:bg-foreground/[0.04]'}"
						>
							{page}
						</button>
					{/each}
					<button
						onclick={() => goToPage(meta.page + 1)}
						disabled={meta.page >= meta.pages}
						class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
					>
						<ChevronRight class="w-4 h-4" />
					</button>
				</div>
			</div>
		{/if}
	{/if}
</div>

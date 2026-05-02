<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { GitBranch, RefreshCw, Search, AlertCircle, Copy, Check } from 'lucide-svelte';
	import { traces as tracesApi } from '$lib/api';
	import type { TraceSummary } from '$lib/types';

	type TimeRange = '1h' | '6h' | '24h';
	type AutoRefresh = 'off' | '15s' | '30s' | '60s';

	let summaries = $state<TraceSummary[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let timeRange = $state<TimeRange>('1h');
	let serviceFilter = $state('');
	let errorsOnly = $state(false);
	let autoRefresh = $state<AutoRefresh>('off');

	let copiedTraceID = $state<string | null>(null);
	let copyResetTimer: ReturnType<typeof setTimeout> | null = null;
	let refreshTimer: ReturnType<typeof setInterval> | null = null;

	const timeTabs: { value: TimeRange; label: string; lookbackMs: number }[] = [
		{ value: '1h', label: 'Last 1h', lookbackMs: 60 * 60 * 1000 },
		{ value: '6h', label: 'Last 6h', lookbackMs: 6 * 60 * 60 * 1000 },
		{ value: '24h', label: 'Last 24h', lookbackMs: 24 * 60 * 60 * 1000 }
	];

	const refreshOptions: { value: AutoRefresh; label: string; intervalMs: number }[] = [
		{ value: 'off', label: 'Off', intervalMs: 0 },
		{ value: '15s', label: '15s', intervalMs: 15_000 },
		{ value: '30s', label: '30s', intervalMs: 30_000 },
		{ value: '60s', label: '60s', intervalMs: 60_000 }
	];

	let filtered = $derived.by(() => {
		if (!errorsOnly) return summaries;
		return summaries.filter((t) => t.has_error);
	});

	function lookbackForRange(r: TimeRange): number {
		return timeTabs.find((t) => t.value === r)?.lookbackMs ?? 60 * 60 * 1000;
	}

	function sinceForRange(r: TimeRange): string {
		return new Date(Date.now() - lookbackForRange(r)).toISOString();
	}

	async function loadData() {
		loadError = null;
		try {
			const resp = await tracesApi.listTraces({
				since: sinceForRange(timeRange),
				service: serviceFilter.trim() || undefined,
				limit: 200
			});
			summaries = resp.data ?? [];
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load traces';
			summaries = [];
		} finally {
			loading = false;
		}
	}

	function applyFilters() {
		loading = true;
		void loadData();
	}

	function startAutoRefresh() {
		if (refreshTimer) {
			clearInterval(refreshTimer);
			refreshTimer = null;
		}
		const opt = refreshOptions.find((o) => o.value === autoRefresh);
		if (opt && opt.intervalMs > 0) {
			refreshTimer = setInterval(() => void loadData(), opt.intervalMs);
		}
	}

	$effect(() => {
		// Reload whenever the time range or service filter changes.
		void timeRange;
		void serviceFilter;
		applyFilters();
	});

	$effect(() => {
		void autoRefresh;
		startAutoRefresh();
	});

	function shortHex(hex: string): string {
		return hex.length > 8 ? hex.slice(0, 8) : hex;
	}

	async function copyTraceID(hex: string) {
		try {
			await navigator.clipboard.writeText(hex);
			copiedTraceID = hex;
			if (copyResetTimer) clearTimeout(copyResetTimer);
			copyResetTimer = setTimeout(() => {
				copiedTraceID = null;
			}, 1000);
		} catch {
			// Clipboard not available — silently noop. The user can still
			// drag-select the visible 8-char prefix.
		}
	}

	function formatDuration(ns: number): string {
		if (ns < 1_000) return `${ns}ns`;
		if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
		if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
		return `${(ns / 1_000_000_000).toFixed(2)}s`;
	}

	// Color buckets: ≤100ms muted, 100ms–1s normal, ≥1s warn, errors override.
	function durationClass(ns: number, hasError: boolean): string {
		if (hasError) return 'text-red-400';
		if (ns >= 1_000_000_000) return 'text-amber-400';
		if (ns >= 100_000_000) return 'text-foreground';
		return 'text-muted-foreground';
	}

	function relativeTime(iso: string): string {
		const then = new Date(iso).getTime();
		const diffMs = Date.now() - then;
		if (diffMs < 0) return 'in the future';
		const sec = Math.floor(diffMs / 1000);
		if (sec < 60) return `${sec}s ago`;
		const min = Math.floor(sec / 60);
		if (min < 60) return `${min}m ${sec % 60}s ago`;
		const hr = Math.floor(min / 60);
		return `${hr}h ${min % 60}m ago`;
	}

	onMount(() => {
		loadData();
	});

	onDestroy(() => {
		if (refreshTimer) clearInterval(refreshTimer);
		if (copyResetTimer) clearTimeout(copyResetTimer);
	});
</script>

<svelte:head>
	<title>Traces - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up">
	<!-- Page header -->
	<div class="flex items-center justify-between mb-5">
		<div>
			<h1 class="text-lg font-semibold text-foreground">Traces</h1>
			<p class="text-xs text-muted-foreground mt-0.5">Distributed traces ingested via OTLP</p>
		</div>
		<button
			onclick={() => applyFilters()}
			class="flex items-center space-x-1.5 px-3 py-2 bg-muted/50 hover:bg-muted text-xs font-medium rounded-md transition-colors text-foreground"
			aria-label="Refresh"
		>
			<RefreshCw class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" />
			<span>Refresh</span>
		</button>
	</div>

	<!-- Filter bar -->
	<div class="flex flex-col sm:flex-row sm:items-center gap-3 mb-4">
		<div class="flex items-center gap-1">
			{#each timeTabs as t}
				<button
					onclick={() => { timeRange = t.value; }}
					class="px-2.5 py-1 text-xs rounded-md transition-colors {timeRange === t.value
						? 'bg-foreground/[0.08] text-foreground font-medium'
						: 'text-muted-foreground hover:text-foreground hover:bg-foreground/[0.04]'}"
				>
					{t.label}
				</button>
			{/each}
		</div>

		<div class="flex items-center gap-2 sm:ml-auto">
			<div class="relative">
				<Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground/50" />
				<input
					type="text"
					bind:value={serviceFilter}
					placeholder="Filter by service..."
					class="w-44 sm:w-56 pl-8 pr-3 py-1.5 bg-card border border-border rounded-md text-xs text-foreground placeholder-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-ring font-mono"
				/>
			</div>

			<label class="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground cursor-pointer select-none">
				<input
					type="checkbox"
					bind:checked={errorsOnly}
					class="w-3.5 h-3.5 rounded border-border bg-card focus:ring-1 focus:ring-ring accent-red-400"
				/>
				<span>Errors only</span>
			</label>

			<select
				bind:value={autoRefresh}
				class="text-xs bg-card border border-border rounded-md px-2 py-1.5 text-muted-foreground hover:text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
				aria-label="Auto-refresh"
			>
				{#each refreshOptions as r}
					<option value={r.value}>↻ {r.label}</option>
				{/each}
			</select>
		</div>
	</div>

	{#if loading}
		<!-- Skeleton -->
		<div class="bg-card border border-border rounded-lg overflow-hidden">
			{#each Array(5) as _}
				<div class="flex items-center px-4 py-3 border-b border-border/20">
					<div class="w-20 h-3 bg-muted/50 rounded animate-pulse"></div>
					<div class="ml-6 w-24 h-3 bg-muted/30 rounded animate-pulse"></div>
					<div class="ml-auto w-16 h-3 bg-muted/50 rounded animate-pulse"></div>
					<div class="ml-4 w-20 h-3 bg-muted/30 rounded animate-pulse"></div>
				</div>
			{/each}
		</div>
	{:else if loadError}
		<div class="bg-card border border-border rounded-lg">
			<div class="p-12 text-center">
				<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
					<AlertCircle class="w-6 h-6 text-red-400/70" />
				</div>
				<p class="text-sm font-medium text-foreground mb-1">Couldn't load traces</p>
				<p class="text-xs text-muted-foreground mb-4 font-mono">{loadError}</p>
				<button
					onclick={() => applyFilters()}
					class="inline-flex items-center space-x-1.5 px-4 py-2 bg-muted/50 hover:bg-muted text-xs font-medium rounded-md transition-colors text-foreground"
				>
					<RefreshCw class="w-3.5 h-3.5" />
					<span>Try again</span>
				</button>
			</div>
		</div>
	{:else if filtered.length === 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="p-12 text-center">
				<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
					<GitBranch class="w-6 h-6 text-muted-foreground/40" />
				</div>
				<p class="text-sm font-medium text-foreground mb-1">
					{summaries.length === 0 ? 'No traces ingested in this window' : 'No matches'}
				</p>
				{#if summaries.length === 0}
					<p class="text-xs text-muted-foreground mb-1">
						Configure your OTLP exporter to point at this Hub.
					</p>
					<div class="inline-block text-left mt-3 bg-muted/30 border border-border rounded-md px-4 py-3 text-[11px] font-mono text-muted-foreground space-y-1">
						<div><span class="text-muted-foreground/60">endpoint</span> &nbsp; <span class="text-foreground">/v1/traces</span></div>
						<div><span class="text-muted-foreground/60">headers</span> &nbsp; <span class="text-foreground">Authorization: Bearer wd_…</span></div>
						<div><span class="text-muted-foreground/60">scope</span> &nbsp;&nbsp;&nbsp;<span class="text-foreground">telemetry_ingest</span></div>
					</div>
				{:else}
					<p class="text-xs text-muted-foreground">
						No traces match the current filter. Try widening the time range or clearing the service filter.
					</p>
				{/if}
			</div>
		</div>
	{:else}
		<div class="bg-card border border-border rounded-lg overflow-hidden">
			<!-- Column headers -->
			<div class="flex items-center px-4 py-2 border-b border-border/30 text-[9px] font-medium text-muted-foreground uppercase tracking-wider">
				<div class="w-32 shrink-0">Trace</div>
				<div class="w-20 shrink-0 text-right ml-auto">Spans</div>
				<div class="w-24 shrink-0 text-right ml-3 hidden sm:block">Duration</div>
				<div class="w-32 shrink-0 text-right ml-3 hidden md:block">Started</div>
				<div class="w-6 shrink-0 ml-2"></div>
			</div>

			<div class="divide-y divide-border/20 font-mono">
				{#each filtered as t (t.trace_id)}
					<a
						href="/traces/{t.trace_id}"
						class="flex items-center px-4 py-3 hover:bg-card-elevated transition-colors group"
					>
						<!-- trace_id (mono, abbreviated) + copy button -->
						<div class="w-32 shrink-0 flex items-center gap-1.5">
							<span class="text-xs text-foreground tabular-nums">{shortHex(t.trace_id)}</span>
							<button
								onclick={(e) => { e.preventDefault(); e.stopPropagation(); void copyTraceID(t.trace_id); }}
								class="opacity-0 group-hover:opacity-100 text-muted-foreground/60 hover:text-foreground transition-all"
								aria-label="Copy trace_id"
								title="Copy trace_id"
							>
								{#if copiedTraceID === t.trace_id}
									<Check class="w-3 h-3 text-emerald-400" />
								{:else}
									<Copy class="w-3 h-3" />
								{/if}
							</button>
						</div>

						<!-- spans -->
						<div class="w-20 shrink-0 text-right ml-auto tabular-nums text-xs text-muted-foreground">
							{t.span_count}
						</div>

						<!-- duration -->
						<div class="w-24 shrink-0 text-right ml-3 hidden sm:block tabular-nums text-xs {durationClass(t.duration_ns, t.has_error)}">
							{formatDuration(t.duration_ns)}
						</div>

						<!-- started -->
						<div class="w-32 shrink-0 text-right ml-3 hidden md:block tabular-nums text-xs text-muted-foreground/80">
							{relativeTime(t.start_time)}
						</div>

						<!-- error dot -->
						<div class="w-6 shrink-0 ml-2 flex justify-end">
							{#if t.has_error}
								<span
									class="w-1.5 h-1.5 rounded-full bg-red-400 shadow-[0_0_4px_rgba(239,68,68,0.6)]"
									aria-label="Has errors"
								></span>
							{/if}
						</div>
					</a>
				{/each}
			</div>

			<div class="px-4 py-2 border-t border-border/30 text-[10px] text-muted-foreground/70 font-mono flex justify-between">
				<span>{filtered.length} trace{filtered.length === 1 ? '' : 's'}</span>
				{#if errorsOnly && summaries.length > filtered.length}
					<span>{summaries.length - filtered.length} hidden by errors-only filter</span>
				{/if}
			</div>
		</div>
	{/if}
</div>

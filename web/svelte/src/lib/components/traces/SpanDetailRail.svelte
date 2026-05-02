<script lang="ts">
	import { X, Copy, Check } from 'lucide-svelte';
	import type { LogRecord, Span } from '$lib/types';
	import { logs as logsApi } from '$lib/api';
	import { serviceChipClass } from '$lib/utils/serviceColors';

	interface Props {
		span: Span;
		traceStart: number; // unix ms
		onClose: () => void;
	}

	let { span, traceStart, onClose }: Props = $props();

	type LogScope = 'span' | 'trace';
	let logScope = $state<LogScope>('span');
	let logsLoading = $state(true);
	let correlatedLogs = $state<LogRecord[]>([]);
	let logsError = $state<string | null>(null);

	let copiedId = $state<string | null>(null);
	let copyTimer: ReturnType<typeof setTimeout> | null = null;

	let spanStartMs = $derived(new Date(span.start_time).getTime());
	let offsetFromStartMs = $derived(spanStartMs - traceStart);
	let durationMs = $derived(span.duration_ns / 1_000_000);
	let endOffsetMs = $derived(offsetFromStartMs + durationMs);

	let kindLabel = $derived(spanKindLabel(span.kind));
	let statusLabel = $derived(spanStatusLabel(span.status_code));
	let isError = $derived(span.status_code === 2);

	// (Re)load logs whenever the span or scope changes. Logs API uses
	// trace_id alone for "whole trace" or trace_id + span_id for "span
	// only" — both filters live in CE PR #138.
	$effect(() => {
		void span.span_id;
		void logScope;
		void loadLogs();
	});

	async function loadLogs() {
		logsLoading = true;
		logsError = null;
		try {
			const params = {
				trace_id: span.trace_id,
				span_id: logScope === 'span' ? span.span_id : undefined,
				// Logs default to a 1h window; widen to 24h so trace-correlated
				// logs from earlier in the day still surface.
				since: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
				limit: 200
			};
			const resp = await logsApi.listLogs(params);
			correlatedLogs = resp.data ?? [];
		} catch (err) {
			logsError = err instanceof Error ? err.message : 'Failed to load logs';
			correlatedLogs = [];
		} finally {
			logsLoading = false;
		}
	}

	function spanKindLabel(k: number): string {
		switch (k) {
			case 1: return 'INTERNAL';
			case 2: return 'SERVER';
			case 3: return 'CLIENT';
			case 4: return 'PRODUCER';
			case 5: return 'CONSUMER';
			default: return 'UNSPECIFIED';
		}
	}

	function spanStatusLabel(c: number): string {
		switch (c) {
			case 1: return 'OK';
			case 2: return 'ERROR';
			default: return 'UNSET';
		}
	}

	function formatDuration(ns: number): string {
		if (ns < 1_000) return `${ns}ns`;
		if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
		if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
		return `${(ns / 1_000_000_000).toFixed(2)}s`;
	}

	function formatOffset(ms: number): string {
		if (ms === 0) return '+0';
		const sign = ms >= 0 ? '+' : '';
		if (Math.abs(ms) < 1) return `${sign}${ms.toFixed(2)}ms`;
		if (Math.abs(ms) < 1000) return `${sign}${ms.toFixed(1)}ms`;
		return `${sign}${(ms / 1000).toFixed(2)}s`;
	}

	function severityClass(text: string): string {
		const t = text.toUpperCase();
		if (t === 'ERROR' || t === 'FATAL' || t === 'CRITICAL') return 'text-red-400';
		if (t === 'WARN' || t === 'WARNING') return 'text-amber-400';
		if (t === 'INFO') return 'text-foreground';
		return 'text-muted-foreground';
	}

	function formatLogTime(iso: string): string {
		const d = new Date(iso);
		const hh = String(d.getHours()).padStart(2, '0');
		const mm = String(d.getMinutes()).padStart(2, '0');
		const ss = String(d.getSeconds()).padStart(2, '0');
		const ms = String(d.getMilliseconds()).padStart(3, '0');
		return `${hh}:${mm}:${ss}.${ms}`;
	}

	function jsonPretty(value: unknown): string {
		if (value == null) return '';
		try {
			return JSON.stringify(value, null, 2);
		} catch {
			return String(value);
		}
	}

	async function copyId(id: string) {
		try {
			await navigator.clipboard.writeText(id);
			copiedId = id;
			if (copyTimer) clearTimeout(copyTimer);
			copyTimer = setTimeout(() => (copiedId = null), 1000);
		} catch {
			// noop — user can still drag-select the visible text
		}
	}

	let attributesText = $derived(jsonPretty(span.attributes));
	let resourceText = $derived(jsonPretty(span.resource));
	let eventsText = $derived(jsonPretty(span.events));
	let hasAttributes = $derived(attributesText.length > 0 && attributesText !== 'null');
	let hasResource = $derived(resourceText.length > 0 && resourceText !== 'null');
	let hasEvents = $derived(eventsText.length > 0 && eventsText !== 'null' && eventsText !== '[]');
</script>

<aside class="bg-card border border-border rounded-lg flex flex-col max-h-[calc(100vh-9rem)] overflow-hidden font-mono">
	<header class="px-4 py-3 border-b border-border/30 flex items-start justify-between gap-3">
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-2 flex-wrap">
				<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[9px] uppercase tracking-wider {serviceChipClass(span.service_name)}">
					{span.service_name}
				</span>
				<span class="text-sm text-foreground truncate">{span.name}</span>
			</div>
			<div class="text-[10px] text-muted-foreground/80 mt-1.5 space-y-0.5">
				<div class="flex items-center gap-1.5">
					<span class="text-muted-foreground/60">span</span>
					<span class="tabular-nums">{span.span_id}</span>
					<button
						onclick={() => copyId(span.span_id)}
						class="text-muted-foreground/60 hover:text-foreground transition-colors"
						aria-label="Copy span_id"
					>
						{#if copiedId === span.span_id}
							<Check class="w-3 h-3 text-emerald-400" />
						{:else}
							<Copy class="w-3 h-3" />
						{/if}
					</button>
				</div>
				{#if span.parent_span_id}
					<div class="flex items-center gap-1.5">
						<span class="text-muted-foreground/60">parent</span>
						<span class="tabular-nums">{span.parent_span_id}</span>
						<button
							onclick={() => copyId(span.parent_span_id!)}
							class="text-muted-foreground/60 hover:text-foreground transition-colors"
							aria-label="Copy parent span_id"
						>
							{#if copiedId === span.parent_span_id}
								<Check class="w-3 h-3 text-emerald-400" />
							{:else}
								<Copy class="w-3 h-3" />
							{/if}
						</button>
					</div>
				{/if}
				<div class="flex items-center gap-1.5">
					<span class="text-muted-foreground/60">kind</span>
					<span>{kindLabel}</span>
				</div>
			</div>
		</div>
		<button
			onclick={onClose}
			class="p-1 rounded hover:bg-muted/50 text-muted-foreground/60 hover:text-foreground transition-colors shrink-0"
			aria-label="Close span detail"
		>
			<X class="w-4 h-4" />
		</button>
	</header>

	<div class="flex-1 overflow-y-auto">
		{#if isError}
			<div class="px-4 py-3 border-b border-border/20 bg-red-400/[0.05]">
				<div class="text-[10px] uppercase tracking-wider text-red-400 font-medium">● Error</div>
				{#if span.status_message}
					<div class="text-xs text-red-300 mt-1">{span.status_message}</div>
				{/if}
			</div>
		{:else}
			<div class="px-4 py-2 border-b border-border/20">
				<div class="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">Status</div>
				<div class="text-xs text-muted-foreground mt-0.5">{statusLabel}</div>
			</div>
		{/if}

		<section class="px-4 py-3 border-b border-border/20">
			<h3 class="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium mb-2">Timing</h3>
			<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
				<dt class="text-muted-foreground/60">start</dt>
				<dd class="tabular-nums text-foreground">{formatOffset(offsetFromStartMs)} <span class="text-muted-foreground/60">from trace start</span></dd>
				<dt class="text-muted-foreground/60">duration</dt>
				<dd class="tabular-nums text-foreground">{formatDuration(span.duration_ns)}</dd>
				<dt class="text-muted-foreground/60">end</dt>
				<dd class="tabular-nums text-foreground">{formatOffset(endOffsetMs)}</dd>
			</dl>
		</section>

		{#if hasAttributes}
			<section class="px-4 py-3 border-b border-border/20">
				<h3 class="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium mb-2">Attributes</h3>
				<pre class="text-[11px] leading-relaxed text-foreground/90 whitespace-pre-wrap break-words">{attributesText}</pre>
			</section>
		{/if}

		{#if hasResource}
			<section class="px-4 py-3 border-b border-border/20">
				<details>
					<summary class="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium mb-2 cursor-pointer hover:text-foreground transition-colors select-none">Resource</summary>
					<pre class="text-[11px] leading-relaxed text-foreground/90 whitespace-pre-wrap break-words mt-2">{resourceText}</pre>
				</details>
			</section>
		{/if}

		{#if hasEvents}
			<section class="px-4 py-3 border-b border-border/20">
				<h3 class="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium mb-2">Events</h3>
				<pre class="text-[11px] leading-relaxed text-foreground/90 whitespace-pre-wrap break-words">{eventsText}</pre>
			</section>
		{/if}

		<section class="px-4 py-3">
			<div class="flex items-center justify-between mb-2">
				<h3 class="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
					Logs in this trace · {correlatedLogs.length}
				</h3>
				<select
					bind:value={logScope}
					class="text-[10px] bg-card border border-border rounded px-1.5 py-0.5 text-muted-foreground hover:text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
					aria-label="Log scope"
				>
					<option value="span">span only</option>
					<option value="trace">whole trace</option>
				</select>
			</div>

			{#if logsLoading}
				<div class="space-y-1">
					{#each Array(3) as _}
						<div class="h-8 bg-muted/30 rounded animate-pulse"></div>
					{/each}
				</div>
			{:else if logsError}
				<div class="text-xs text-muted-foreground/70 italic">{logsError}</div>
			{:else if correlatedLogs.length === 0}
				<div class="text-xs text-muted-foreground/60 italic">
					No logs correlated to this {logScope === 'span' ? 'span' : 'trace'}.
				</div>
			{:else}
				<ul class="space-y-1.5 text-[11px]">
					{#each correlatedLogs as l}
						<li class="border-l-2 border-border/40 pl-2">
							<div class="flex items-center gap-2 text-muted-foreground/70">
								<span class="tabular-nums">{formatLogTime(l.timestamp)}</span>
								<span class="{severityClass(l.severity_text ?? '')} uppercase tracking-wider text-[9px]">
									{l.severity_text ?? '—'}
								</span>
								<span class="text-muted-foreground/50">{l.service_name}</span>
							</div>
							<div class="text-foreground/90 mt-0.5 break-words">{l.body}</div>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	</div>
</aside>

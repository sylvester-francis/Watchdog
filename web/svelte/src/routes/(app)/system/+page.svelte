<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';

	import { KeyRound, Copy, Check, Trash2 } from 'lucide-svelte';
	import { Skeleton } from '@sylvester-francis/watchdog-ui';
	import { system as systemApi } from '$lib/api';
	import { getAuth } from '$lib/stores/auth.svelte';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { SystemInfo, AdminUser, MetricsResponse, MetricsSnapshot } from '$lib/types';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import {
		Chart,
		LineController,
		LineElement,
		PointElement,
		LinearScale,
		CategoryScale,
		Filler,
		Tooltip,
		Legend
	} from 'chart.js';

	Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Filler, Tooltip, Legend);

	const auth = getAuth();
	const toast = getToasts();
	const isAdmin = $derived(auth.user?.is_admin === true);

	let data = $state<SystemInfo | null>(null);
	let users = $state<AdminUser[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Hub Metrics state
	let metrics = $state<MetricsResponse | null>(null);
	let metricsInterval: ReturnType<typeof setInterval> | null = null;
	let httpLatencyCanvas = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let heartbeatCanvas = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let requestRateCanvas = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let httpLatencyChart: Chart | null = null;
	let heartbeatChart: Chart | null = null;
	let requestRateChart: Chart | null = null;

	const chartTooltip = {
		backgroundColor: '#18181b',
		borderColor: '#27272a',
		borderWidth: 1,
		titleFont: { family: "'Inter', system-ui, sans-serif" as const, size: 11 },
		bodyFont: { family: "'JetBrains Mono', ui-monospace, monospace" as const, size: 12 },
		padding: 10,
		cornerRadius: 6,
		displayColors: true,
		boxWidth: 8,
		boxPadding: 4
	};

	const chartScaleDefaults = {
		x: {
			grid: { color: '#27272a', lineWidth: 0.5 },
			ticks: { maxTicksLimit: 8, maxRotation: 0, color: '#71717a', font: { size: 10, family: "'JetBrains Mono', ui-monospace, monospace" as const } },
			border: { color: '#27272a' }
		},
		y: {
			grid: { color: '#27272a', lineWidth: 0.5 },
			ticks: { color: '#71717a', font: { size: 10, family: "'JetBrains Mono', ui-monospace, monospace" as const } },
			beginAtZero: true,
			border: { color: '#27272a' }
		}
	};

	function formatChartTime(ts: number): string {
		const d = new Date(ts * 1000);
		return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function buildMetricsCharts() {
		const history = metrics?.history;
		if (!history || history.length < 2) return;

		const labels = history.map(s => formatChartTime(s.timestamp));

		if (httpLatencyCanvas) {
			if (httpLatencyChart) httpLatencyChart.destroy();
			httpLatencyChart = new Chart(httpLatencyCanvas, {
				type: 'line',
				data: {
					labels,
					datasets: [
						{ label: 'p50', data: history.map(s => s.http_latency_p50), borderColor: '#3b82f6', borderWidth: 2, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false },
						{ label: 'p95', data: history.map(s => s.http_latency_p95), borderColor: '#fbbf24', borderWidth: 1.5, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false },
						{ label: 'p99', data: history.map(s => s.http_latency_p99), borderColor: '#f87171', borderWidth: 1, borderDash: [3, 3], pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false }
					]
				},
				options: {
					responsive: true, maintainAspectRatio: false,
					interaction: { mode: 'index', intersect: false },
					plugins: {
						legend: { display: true, position: 'top', align: 'end', labels: { boxWidth: 12, padding: 10, font: { size: 10 }, color: '#a1a1aa', usePointStyle: true, pointStyle: 'line' } },
						tooltip: { ...chartTooltip, callbacks: { label: (ctx: any) => ` ${ctx.dataset.label}: ${ctx.parsed.y}ms` } }
					},
					scales: { ...chartScaleDefaults, y: { ...chartScaleDefaults.y, ticks: { ...chartScaleDefaults.y.ticks, callback: (v: any) => `${v}ms` } } },
					animation: { duration: 0 }
				}
			});
		}

		if (heartbeatCanvas) {
			if (heartbeatChart) heartbeatChart.destroy();
			heartbeatChart = new Chart(heartbeatCanvas, {
				type: 'line',
				data: {
					labels,
					datasets: [
						{ label: 'p50', data: history.map(s => s.heartbeat_p50), borderColor: '#3b82f6', borderWidth: 2, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false },
						{ label: 'p95', data: history.map(s => s.heartbeat_p95), borderColor: '#fbbf24', borderWidth: 1.5, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false }
					]
				},
				options: {
					responsive: true, maintainAspectRatio: false,
					interaction: { mode: 'index', intersect: false },
					plugins: {
						legend: { display: true, position: 'top', align: 'end', labels: { boxWidth: 12, padding: 10, font: { size: 10 }, color: '#a1a1aa', usePointStyle: true, pointStyle: 'line' } },
						tooltip: { ...chartTooltip, callbacks: { label: (ctx: any) => ` ${ctx.dataset.label}: ${ctx.parsed.y}ms` } }
					},
					scales: { ...chartScaleDefaults, y: { ...chartScaleDefaults.y, ticks: { ...chartScaleDefaults.y.ticks, callback: (v: any) => `${v}ms` } } },
					animation: { duration: 0 }
				}
			});
		}

		if (requestRateCanvas) {
			if (requestRateChart) requestRateChart.destroy();
			requestRateChart = new Chart(requestRateCanvas, {
				type: 'line',
				data: {
					labels,
					datasets: [
						{ label: 'req/s', data: history.map(s => s.http_request_rate), borderColor: '#3b82f6', backgroundColor: 'rgba(59, 130, 246, 0.08)', borderWidth: 2, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: true }
					]
				},
				options: {
					responsive: true, maintainAspectRatio: false,
					interaction: { mode: 'index', intersect: false },
					plugins: {
						legend: { display: false },
						tooltip: { ...chartTooltip, callbacks: { label: (ctx: any) => ` ${ctx.parsed.y} req/s` } }
					},
					scales: { ...chartScaleDefaults, y: { ...chartScaleDefaults.y, ticks: { ...chartScaleDefaults.y.ticks, callback: (v: any) => `${v}/s` } } },
					animation: { duration: 0 }
				}
			});
		}
	}

	async function refreshMetrics() {
		try {
			metrics = await systemApi.getMetrics();
			buildMetricsCharts();
		} catch {
			// Silently fail — metrics are supplemental
		}
	}

	function destroyCharts() {
		httpLatencyChart?.destroy(); httpLatencyChart = null;
		heartbeatChart?.destroy(); heartbeatChart = null;
		requestRateChart?.destroy(); requestRateChart = null;
	}

	// Password reset state
	let resetPassword = $state('');
	let resetUserEmail = $state('');
	let copiedPassword = $state(false);
	let confirmModal = $state<{
		open: boolean;
		title: string;
		message: string;
		confirmLabel: string;
		variant: 'danger' | 'warning';
		loading: boolean;
		action: (() => Promise<void>) | null;
	}>({
		open: false, title: '', message: '', confirmLabel: 'Confirm', variant: 'warning', loading: false, action: null
	});

	function closeConfirmModal() {
		confirmModal = { open: false, title: '', message: '', confirmLabel: 'Confirm', variant: 'warning', loading: false, action: null };
	}

	async function executeConfirm() {
		if (confirmModal.action) await confirmModal.action();
	}

	function handleResetPassword(user: AdminUser) {
		confirmModal = {
			open: true,
			title: 'Reset Password',
			message: `Reset the password for ${user.email}? They will be required to change it on next login.`,
			confirmLabel: 'Reset Password',
			variant: 'warning',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					const res = await systemApi.resetUserPassword(user.id);
					resetPassword = res.password;
					resetUserEmail = user.email;
					closeConfirmModal();
				} catch (err) {
					toast.error(err instanceof Error ? err.message : 'Failed to reset password.');
					confirmModal.loading = false;
				}
			}
		};
	}

	function handleDeleteUser(user: AdminUser) {
		confirmModal = {
			open: true,
			title: 'Delete User',
			message: `Permanently delete ${user.email}? This will remove their agents, monitors, and all associated data. This cannot be undone.`,
			confirmLabel: 'Delete User',
			variant: 'danger',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					await systemApi.deleteUser(user.id);
					users = users.filter(u => u.id !== user.id);
					toast.success(`User ${user.email} deleted.`);
					closeConfirmModal();
				} catch (err) {
					toast.error(err instanceof Error ? err.message : 'Failed to delete user.');
					confirmModal.loading = false;
				}
			}
		};
	}

	async function copyPasswordToClipboard() {
		await navigator.clipboard.writeText(resetPassword);
		copiedPassword = true;
		setTimeout(() => { copiedPassword = false; }, 2000);
	}

	function dismissResetPassword() {
		resetPassword = '';
		resetUserEmail = '';
		copiedPassword = false;
	}

	function timeAgo(dateStr: string): string {
		const diff = Date.now() - new Date(dateStr).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		return `${days}d ago`;
	}

	function actionTextClass(action: string): string {
		if (action === 'login_success') return 'text-success';
		if (action === 'login_failed' || action.endsWith('_deleted') || action.endsWith('_revoked')) return 'text-destructive';
		if (action.endsWith('_updated') || action.startsWith('incident_')) return 'text-warning';
		return 'text-muted-foreground';
	}

	onMount(async () => {
		try {
			const [sysResult, usersResult, metricsResult] = await Promise.allSettled([
				systemApi.getSystemInfo(),
				systemApi.listUsers(),
				systemApi.getMetrics()
			]);

			if (sysResult.status === 'fulfilled') {
				data = sysResult.value;
			} else {
				throw sysResult.reason;
			}

			if (usersResult.status === 'fulfilled') {
				users = usersResult.value.data ?? [];
			}

			if (metricsResult.status === 'fulfilled') {
				metrics = metricsResult.value;
			}

			metricsInterval = setInterval(refreshMetrics, 10000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load system info';
		} finally {
			loading = false;
			await tick();
			buildMetricsCharts();
		}
	});

	onDestroy(() => {
		if (metricsInterval) clearInterval(metricsInterval);
		destroyCharts();
	});
</script>

<svelte:head>
	<title>System - WatchDog</title>
</svelte:head>

{#if loading}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] space-y-6 px-4 py-6 sm:px-6 sm:py-10">
		<Skeleton emphasis="secondary" width="6rem" height="1.75rem" />
		<div class="grid grid-cols-2 gap-px border-y border-border bg-border lg:grid-cols-4">
			{#each Array(4) as _}
				<div class="bg-background p-4">
					<Skeleton emphasis="tertiary" width="4rem" height="0.625rem" />
					<div class="mt-2">
						<Skeleton emphasis="secondary" width="4rem" height="1.25rem" />
					</div>
				</div>
			{/each}
		</div>
		<Skeleton variant="card" emphasis="secondary" height="16rem" />
	</div>
{:else if error}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<p class="text-sm font-medium text-foreground">Failed to load system info</p>
		<p class="mt-1 font-mono tabular-nums text-xs text-destructive">{error}</p>
	</div>
{:else if data}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<header>
			<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
				<span class="uppercase tracking-wider">System</span>
			</div>
			<h1 class="mt-1.5 text-xl font-medium text-foreground sm:text-2xl md:text-3xl">System</h1>
			<p class="mt-1 text-sm text-muted-foreground">Server health, performance metrics, and audit log.</p>
		</header>

		<!-- System Health: hairline-separated stat columns -->
		<div class="mt-8 grid grid-cols-2 gap-px overflow-hidden border-y border-border bg-border lg:grid-cols-4">
			<div class="flex flex-col bg-background px-4 py-3.5">
				<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Database</div>
				<div class="mt-1 font-mono tabular-nums text-lg {data.db.healthy ? 'text-foreground' : 'text-destructive'}">{data.db.ping_ms.toFixed(0)}ms</div>
				<div class="mt-0.5 text-[11px] text-muted-foreground">{data.db.healthy ? 'Healthy' : 'Unreachable'}</div>
			</div>
			<div class="flex flex-col bg-background px-4 py-3.5">
				<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">DB Connections</div>
				<div class="mt-1 font-mono tabular-nums text-lg text-foreground">{data.db.pool.acquired}<span class="text-muted-foreground/60"> / {data.db.pool.total}</span></div>
				<div class="mt-0.5 font-mono tabular-nums text-[11px] text-muted-foreground">{data.db.pool.idle} idle</div>
			</div>
			<div class="flex flex-col bg-background px-4 py-3.5">
				<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Agents Online</div>
				<div class="mt-1 font-mono tabular-nums text-lg {data.agents_connected > 0 ? 'text-success' : 'text-muted-foreground'}">{data.agents_connected}</div>
				<div class="mt-0.5 text-[11px] text-muted-foreground">{data.agents_connected > 0 ? 'Reporting' : 'No agents'}</div>
			</div>
			<div class="flex flex-col bg-background px-4 py-3.5">
				<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Uptime</div>
				<div class="mt-1 font-mono tabular-nums text-lg text-foreground">{data.runtime.uptime_formatted}</div>
				<div class="mt-0.5 text-[11px] text-muted-foreground">Since restart</div>
			</div>
		</div>

		<!-- Hub Metrics (Prometheus) -->
		<section class="mt-10">
			<div class="flex flex-wrap items-baseline justify-between gap-2 border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Hub Metrics</h3>
				<span class="font-mono tabular-nums text-[11px] text-muted-foreground">Auto-refresh · 10s</span>
			</div>
			<p class="pt-3 text-xs text-muted-foreground">Live performance data from the hub server. Use these to spot bottlenecks before they become outages.</p>

			{#if metrics}
				{@const cur = metrics.current}
				<!-- Live Gauges: hairline columns -->
				<div class="mt-4 grid grid-cols-1 gap-px overflow-hidden border-y border-border bg-border sm:grid-cols-3">
					<div class="flex flex-col bg-background px-4 py-3.5">
						<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">WS Connections</div>
						<div class="mt-1 font-mono tabular-nums text-lg text-foreground">{cur.ws_connections}</div>
						<div class="mt-0.5 text-[11px] text-muted-foreground">Agents connected right now</div>
					</div>
					<div class="flex flex-col bg-background px-4 py-3.5">
						<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">DB Pool Active</div>
						<div class="mt-1 font-mono tabular-nums text-lg text-foreground">{cur.db_pool_active}</div>
						<div class="mt-0.5 text-[11px] text-muted-foreground">Database connections in use</div>
					</div>
					<div class="flex flex-col bg-background px-4 py-3.5">
						<div class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Active Incidents</div>
						<div class="mt-1 flex items-baseline gap-2">
							<span class="font-mono tabular-nums text-lg {cur.incidents_open > 0 ? 'text-destructive' : 'text-foreground'}">{cur.incidents_open}</span>
							{#if cur.incidents_acked > 0}
								<span class="font-mono tabular-nums text-[11px] text-muted-foreground">+{cur.incidents_acked} ack'd</span>
							{/if}
						</div>
						<div class="mt-0.5 text-[11px] text-muted-foreground">Unresolved problems</div>
					</div>
				</div>

				<!-- Time-series Charts -->
				{#if metrics.history && metrics.history.length >= 2}
					<div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
						<div>
							<div class="border-b border-border pb-2">
								<h4 class="text-sm font-medium text-foreground">HTTP Latency</h4>
								<p class="mt-0.5 text-xs text-muted-foreground">How fast the hub responds to API requests. Lower is better.</p>
							</div>
							<div class="h-[200px] pt-3">
								<canvas bind:this={httpLatencyCanvas}></canvas>
							</div>
						</div>
						<div>
							<div class="border-b border-border pb-2">
								<h4 class="text-sm font-medium text-foreground">Heartbeat Processing</h4>
								<p class="mt-0.5 text-xs text-muted-foreground">Time to process each health check from agents. Spikes mean the hub is overloaded.</p>
							</div>
							<div class="h-[200px] pt-3">
								<canvas bind:this={heartbeatCanvas}></canvas>
							</div>
						</div>
						<div class="lg:col-span-2">
							<div class="border-b border-border pb-2">
								<h4 class="text-sm font-medium text-foreground">Request Rate</h4>
								<p class="mt-0.5 text-xs text-muted-foreground">Total API requests per second hitting the hub. Helps size capacity and spot traffic anomalies.</p>
							</div>
							<div class="h-[180px] pt-3">
								<canvas bind:this={requestRateCanvas}></canvas>
							</div>
						</div>
					</div>
				{:else}
					<p class="pt-4 text-xs text-muted-foreground">Collecting data… charts will appear after ~20s of history.</p>
				{/if}
			{/if}
		</section>

		<!-- Operational Metrics -->
		<div class="mt-10 grid grid-cols-1 gap-10 lg:grid-cols-2">
			<!-- Heartbeat Throughput -->
			<section>
				<div class="border-b border-border pb-3">
					<h3 class="text-sm font-medium text-foreground">Heartbeat Throughput</h3>
				</div>
				<div class="divide-y divide-border/40">
					<div class="flex items-center justify-between py-3">
						<span class="text-xs text-muted-foreground">Checks per minute</span>
						<span class="font-mono tabular-nums text-sm text-foreground">{data.heartbeats.per_minute.toFixed(1)}/min</span>
					</div>
					<div class="flex items-center justify-between py-3">
						<span class="text-xs text-muted-foreground">Total checks in last hour</span>
						<span class="font-mono tabular-nums text-sm text-foreground">{data.heartbeats.total_last_hour}</span>
					</div>
					<div class="flex items-center justify-between py-3">
						<span class="text-xs text-muted-foreground">Failed checks in last hour</span>
						<span class="font-mono tabular-nums text-sm {data.heartbeats.errors_last_hour > 0 ? 'text-destructive' : 'text-success'}">
							{data.heartbeats.errors_last_hour}
						</span>
					</div>
				</div>
			</section>

			<!-- Storage & Runtime -->
			<section>
				<div class="border-b border-border pb-3">
					<h3 class="text-sm font-medium text-foreground">Storage & Runtime</h3>
				</div>
				<div class="divide-y divide-border/40">
					<div class="flex items-center justify-between py-3">
						<span class="text-xs text-muted-foreground">Total disk used by database</span>
						<span class="font-mono tabular-nums text-sm text-foreground">{data.db.size}</span>
					</div>
					<div class="flex items-center justify-between py-3">
						<span class="text-xs text-muted-foreground">Active background tasks</span>
						<span class="font-mono tabular-nums text-sm text-foreground">{data.runtime.goroutines}</span>
					</div>
					<div class="flex items-center justify-between py-3">
						<span class="text-xs text-muted-foreground">Memory in use</span>
						<span class="font-mono tabular-nums text-sm text-foreground">{data.runtime.heap_mb} MB</span>
					</div>
					<div class="flex items-center justify-between py-3">
						<span class="text-xs text-muted-foreground">Last garbage collection pause</span>
						<span class="font-mono tabular-nums text-sm text-foreground">{data.runtime.gc_pause_ms} ms</span>
					</div>
				</div>
			</section>
		</div>

		<!-- Table Sizes -->
		{#if data.db.table_sizes.length > 0}
			<section class="mt-10">
				<div class="flex flex-wrap items-baseline justify-between gap-2 border-b border-border pb-3">
					<h3 class="text-sm font-medium text-foreground">Table Sizes</h3>
					<span class="font-mono tabular-nums text-[11px] text-muted-foreground">Largest 5 tables by disk space</span>
				</div>
				<div class="divide-y divide-border/40">
					{#each data.db.table_sizes as table}
						<div class="flex items-center justify-between py-3">
							<span class="font-mono tabular-nums text-xs text-muted-foreground">{table.name}</span>
							<span class="font-mono tabular-nums text-xs text-foreground">{table.size}</span>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Migration Status -->
		<section class="mt-10">
			<div class="border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Migration Status</h3>
			</div>
			<div class="flex flex-wrap items-center gap-x-6 gap-y-2 pt-4 font-mono tabular-nums text-xs text-muted-foreground">
				<span>Schema version <span class="text-foreground">{data.db.migration.version}</span></span>
				<span class="text-muted-foreground/40">·</span>
				<span class="flex items-center gap-1.5">
					Failed migration
					{#if data.db.migration.dirty}
						<span class="text-destructive">Yes</span>
					{:else}
						<span class="text-success">No</span>
					{/if}
				</span>
			</div>
		</section>

		{#if isAdmin}
		<!-- Reset Password Banner (shown after admin reset) -->
		{#if resetPassword}
			<section class="mt-10 border border-warning/40 bg-warning/[0.04] p-4">
				<div class="flex items-start justify-between gap-3">
					<div class="flex items-center gap-2">
						<KeyRound class="h-4 w-4 shrink-0 text-warning" />
						<span class="text-sm font-medium text-foreground">Password Reset</span>
					</div>
					<button
						onclick={dismissResetPassword}
						class="text-xs text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
					>
						Dismiss
					</button>
				</div>
				<p class="mt-2 text-xs text-muted-foreground">
					Temporary password for <span class="font-mono tabular-nums text-foreground">{resetUserEmail}</span>.
					Copy now — you won't see it again. The user will be required to change it on next login.
				</p>
				<div class="mt-3 flex items-center gap-2">
					<code class="flex-1 select-all break-all border border-border bg-background px-3 py-2 font-mono text-xs text-foreground">{resetPassword}</code>
					<button
						onclick={copyPasswordToClipboard}
						class="shrink-0 text-muted-foreground transition-colors hover:text-foreground"
						aria-label="Copy password"
					>
						{#if copiedPassword}
							<Check class="h-4 w-4 text-success" />
						{:else}
							<Copy class="h-4 w-4" />
						{/if}
					</button>
				</div>
			</section>
		{/if}

		<!-- Users -->
		<section class="mt-10">
			<div class="flex items-baseline gap-2 border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Users</h3>
				{#if users.length > 0}
					<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{users.length}</span>
				{/if}
			</div>

			{#if users.length > 0}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="border-b border-border">
								<th class="py-2.5 pl-1 pr-4 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Email</th>
								<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground sm:table-cell">Username</th>
								<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground md:table-cell">Plan</th>
								<th class="hidden px-4 py-2.5 text-right text-[10px] font-medium uppercase tracking-wider text-muted-foreground lg:table-cell">Agents</th>
								<th class="hidden px-4 py-2.5 text-right text-[10px] font-medium uppercase tracking-wider text-muted-foreground lg:table-cell">Monitors</th>
								<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground md:table-cell">Joined</th>
								<th class="px-4 py-2.5 text-right text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border/40">
							{#each users as u (u.id)}
								<tr class="transition-colors hover:bg-muted/30">
									<td class="py-3 pl-1 pr-4">
										<div class="flex items-center gap-2">
											<span class="font-mono tabular-nums text-xs text-foreground">{u.email}</span>
											{#if u.is_admin}
												<span class="font-mono tabular-nums text-[10px] uppercase tracking-wider text-warning">Admin</span>
											{/if}
										</div>
									</td>
									<td class="hidden px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground sm:table-cell">{u.username}</td>
									<td class="hidden px-4 py-3 text-xs capitalize text-muted-foreground md:table-cell">{u.plan}</td>
									<td class="hidden px-4 py-3 text-right font-mono tabular-nums text-xs text-muted-foreground lg:table-cell">{u.agent_count}</td>
									<td class="hidden px-4 py-3 text-right font-mono tabular-nums text-xs text-muted-foreground lg:table-cell">{u.monitor_count}</td>
									<td class="hidden px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground md:table-cell">{timeAgo(u.created_at)}</td>
									<td class="px-4 py-3 text-right">
										{#if u.id !== auth.user?.id}
											<div class="flex items-center justify-end gap-3 text-xs">
												<button
													onclick={() => handleResetPassword(u)}
													class="text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
												>
													Reset
												</button>
												<button
													onclick={() => handleDeleteUser(u)}
													class="flex items-center gap-1 text-destructive underline-offset-4 transition-colors hover:underline"
												>
													<Trash2 class="h-3 w-3" />
													<span>Delete</span>
												</button>
											</div>
										{:else}
											<span class="text-[11px] text-muted-foreground/40">You</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<p class="pt-4 text-xs text-muted-foreground">No users found. Registered users will appear here.</p>
			{/if}
		</section>
		{/if}

		<!-- Audit Log -->
		<section class="mt-10">
			<div class="flex flex-wrap items-baseline justify-between gap-2 border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Audit Log</h3>
				<span class="font-mono tabular-nums text-[11px] text-muted-foreground">Last 50 events</span>
			</div>

			{#if data.audit_logs.length > 0}
				<div class="overflow-x-auto max-h-[32rem] overflow-y-auto">
					<table class="w-full">
						<thead class="sticky top-0 bg-card z-10">
							<tr class="border-b border-border">
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider w-24">Time</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Action</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">User</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden sm:table-cell w-32">IP Address</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">Details</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border/40">
							{#each data.audit_logs as log}
								<tr class="transition-colors hover:bg-muted/30">
									<td class="whitespace-nowrap py-3 pl-1 pr-4 font-mono tabular-nums text-xs text-muted-foreground">{timeAgo(log.created_at)}</td>
									<td class="whitespace-nowrap px-4 py-3 font-mono tabular-nums text-[11px] uppercase tracking-wider {actionTextClass(log.action)}">
										{log.action}
									</td>
									<td class="px-4 py-3 font-mono tabular-nums text-xs text-foreground">
										{log.user_email || '—'}
									</td>
									<td class="hidden px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground sm:table-cell">
										{log.ip_address || '—'}
									</td>
									<td class="hidden max-w-xs truncate px-4 py-3 md:table-cell">
										{#if log.metadata && Object.keys(log.metadata).length > 0}
											<span class="font-mono tabular-nums text-[10px] text-muted-foreground">
												{#each Object.entries(log.metadata) as [k, v]}
													<span class="mr-2"><span class="text-muted-foreground/50">{k}:</span> {v}</span>
												{/each}
											</span>
										{:else}
											<span class="text-xs text-muted-foreground/40">—</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<p class="pt-4 text-xs text-muted-foreground">No audit log entries yet. Events will appear here as users interact with the system.</p>
			{/if}
		</section>
	</div>

	<ConfirmModal
		open={confirmModal.open}
		title={confirmModal.title}
		message={confirmModal.message}
		confirmLabel={confirmModal.confirmLabel}
		variant={confirmModal.variant}
		loading={confirmModal.loading}
		onConfirm={executeConfirm}
		onCancel={closeConfirmModal}
	/>
{/if}
